package repo

import (
	"bytes"
	"cdp/config"
	"cdp/entity"
	"cdp/pkg/goutil"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/go-elasticsearch/v7" // TODO: Using v7.11.0 to be compatible with Bonsai ES
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/elastic/go-elasticsearch/v7/esutil"
	"github.com/rs/zerolog/log"
	"net/http"
	"strings"
	"time"
)

type QueryRepo interface {
	CreateStore(_ context.Context, tenantName string) error
	BatchUpsert(ctx context.Context, tenantName string, udTagVals []*entity.UdTagVal) error
	Count(ctx context.Context, tenantName string, query *entity.Query) (uint64, error)
	Download(ctx context.Context, tenantName string, query *entity.Query, page *Pagination) ([]*entity.Ud, *Pagination, error)
	OnInsertSuccess() chan struct{}
	OnInsertFailure() chan error
	Close(ctx context.Context) error
}

type queryRepo struct {
	client          *elasticsearch.Client
	bulkIndexer     esutil.BulkIndexer
	scrollTimeout   time.Duration
	onInsertSuccess chan struct{}
	onInsertFailure chan error
}

var (
	defaultNumWorkers           = 10
	defaultFlushBytes           = 1_000_000
	defaultFlushIntervalSeconds = 5
	successChanSize             = 500_000
	failureChanSize             = 500_000
)

func NewQueryRepo(_ context.Context, cfg config.ElasticSearch) (QueryRepo, error) {
	retryBackOff := backoff.NewExponentialBackOff()

	c, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses:  cfg.Addr,
		Username:   cfg.Username,
		Password:   cfg.Password,
		MaxRetries: 5,
		RetryBackoff: func(i int) time.Duration {
			if i == 1 {
				retryBackOff.Reset()
			}
			return retryBackOff.NextBackOff()
		},
		RetryOnStatus: []int{
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
			http.StatusTooManyRequests,
		},
	})
	if err != nil {
		return nil, err
	}

	numWorkers := cfg.NumWorkers
	if numWorkers == 0 {
		numWorkers = defaultNumWorkers
	}

	flushBytes := cfg.FlushBytes
	if flushBytes == 0 {
		flushBytes = defaultFlushBytes
	}

	flushIntervalSeconds := cfg.FlushInternalSeconds
	if flushIntervalSeconds == 0 {
		flushIntervalSeconds = defaultFlushIntervalSeconds
	}

	indexer, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client:        c,
		NumWorkers:    numWorkers,
		FlushBytes:    flushBytes,
		FlushInterval: time.Duration(flushIntervalSeconds) * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return &queryRepo{
		client:          c,
		bulkIndexer:     indexer,
		onInsertSuccess: make(chan struct{}, successChanSize),
		onInsertFailure: make(chan error, failureChanSize),
		scrollTimeout:   time.Duration(cfg.ScrollTimeoutSeconds) * time.Second,
	}, nil
}

func (r *queryRepo) CreateStore(ctx context.Context, tenantName string) error {
	mapping := map[string]interface{}{
		"mappings": map[string]interface{}{
			"dynamic_templates": []map[string]interface{}{
				{
					"strings_as_keywords": map[string]interface{}{
						"match_mapping_type": "string",
						"mapping": map[string]interface{}{
							"type": "keyword",
						},
					},
				},
			},
		},
	}

	mappingBody, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("failed to marshal mapping: %w", err)
	}

	res, err := r.client.Indices.Create(
		tenantName,
		r.client.Indices.Create.WithBody(bytes.NewReader(mappingBody)),
		r.client.Indices.Create.WithContext(ctx),
	)
	if err != nil {
		return err
	}

	defer func() {
		_ = res.Body.Close()
	}()

	var createResp map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&createResp); err != nil {
		return err
	}

	if err := r.extractElasticError(createResp); err != nil {
		return err
	}

	return nil
}

func (r *queryRepo) OnInsertSuccess() chan struct{} {
	return r.onInsertSuccess
}

func (r *queryRepo) OnInsertFailure() chan error {
	return r.onInsertFailure
}

// BatchUpsert bulk indexer: https://github.com/elastic/go-elasticsearch/blob/main/_examples/bulk/indexer.go
func (r *queryRepo) BatchUpsert(ctx context.Context, tenantName string, udTagVals []*entity.UdTagVal) error {
	for _, udTagVal := range udTagVals {
		if udTagVal == nil {
			log.Ctx(ctx).Warn().Msg("nil udTagVal found in batch upsert")
			continue
		}

		docID := udTagVal.GetUd().ToDocID()

		if docID == "" {
			log.Ctx(ctx).Warn().Msg("empty doc ID found in batch upsert")
			continue
		}

		data, err := udTagVal.ToDoc()
		if err != nil {
			log.Ctx(ctx).Error().Msgf("fail to convert udTagVal to doc: %v", err)
			return err
		}

		if err := r.bulkIndexer.Add(ctx, esutil.BulkIndexerItem{
			Action:     "update",
			Index:      tenantName,
			DocumentID: docID,
			Body:       strings.NewReader(fmt.Sprintf(`{"doc":%s, "doc_as_upsert": true}`, data)),
			OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem) {
				select {
				case r.onInsertSuccess <- struct{}{}:
				default:
				}
			},
			OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
				if err == nil {
					err = fmt.Errorf("type: %s, reason: %s", res.Error.Type, res.Error.Reason)
				}

				select {
				case r.onInsertFailure <- err:
				default:
				}
			},
		}); err != nil {
			log.Ctx(ctx).Error().Msgf("fail to add udTagVal to indexer: %v, docID: %v, data: %v", err, docID, data)
			return err
		}
	}

	return nil
}

func (r *queryRepo) Download(ctx context.Context, tenantName string, query *entity.Query, page *Pagination) ([]*entity.Ud, *Pagination, error) {
	var (
		res *esapi.Response
		err error
	)
	if page.GetCursor() != "" {
		res, err = r.client.Scroll(
			r.client.Scroll.WithScroll(r.scrollTimeout),
			r.client.Scroll.WithScrollID(page.GetCursor()),
			r.client.Scroll.WithContext(ctx),
		)
	} else {
		queryBody := r.buildElasticQuery(query)
		if queryBody == nil {
			return nil, nil, nil
		}

		body, err := json.Marshal(map[string]interface{}{
			"query":   queryBody,
			"size":    page.GetLimit(),
			"_source": false,
		})
		if err != nil {
			return nil, nil, err
		}

		res, err = r.client.Search(
			r.client.Search.WithIndex(tenantName),
			r.client.Search.WithBody(bytes.NewReader(body)),
			r.client.Search.WithScroll(r.scrollTimeout),
			r.client.Search.WithContext(ctx),
		)
	}
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	var searchResp map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&searchResp); err != nil {
		return nil, nil, err
	}

	if err := r.extractElasticError(searchResp); err != nil {
		return nil, nil, err
	}

	hits, ok := searchResp["hits"].(map[string]interface{})["hits"].([]interface{})
	if !ok {
		return nil, nil, errors.New("no hits found in response")
	}

	uds := make([]*entity.Ud, 0)
	for _, hit := range hits {
		if doc, ok := hit.(map[string]interface{}); ok {
			if id, exists := doc["_id"].(string); exists {
				ud, err := entity.ToUd(id)
				if err != nil {
					return nil, nil, err
				}
				uds = append(uds, ud)
			}
		}
	}

	newPage := &Pagination{
		Limit:  page.Limit,
		Cursor: goutil.String(""),
	}
	if sid, ok := searchResp["_scroll_id"].(string); ok {
		if uint32(len(uds)) >= page.GetLimit() {
			newPage.Cursor = goutil.String(sid)
		} else {
			res, err := r.client.ClearScroll(
				r.client.ClearScroll.WithScrollID(sid),
				r.client.ClearScroll.WithContext(ctx),
			)
			if err != nil || res.IsError() {
				log.Ctx(ctx).Error().Msgf("fail to clear scroll id %s: %v", sid, err)
			}
		}
	}

	return uds, newPage, nil
}

func (r *queryRepo) Count(ctx context.Context, tenantName string, query *entity.Query) (uint64, error) {
	queryBody := r.buildElasticQuery(query)
	if queryBody == nil {
		return 0, nil
	}

	body, err := json.Marshal(map[string]interface{}{"query": queryBody})
	if err != nil {
		return 0, err
	}

	res, err := r.client.Count(
		r.client.Count.WithIndex(tenantName),
		r.client.Count.WithBody(bytes.NewReader(body)),
		r.client.Count.WithContext(ctx),
	)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	var countResp map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&countResp); err != nil {
		return 0, err
	}

	if err := r.extractElasticError(countResp); err != nil {
		return 0, err
	}

	if count, ok := countResp["count"].(float64); ok {
		return uint64(count), nil
	}

	return 0, fmt.Errorf("unexpected response format")
}

func (r *queryRepo) extractElasticError(resp map[string]interface{}) error {
	if errorResp, ok := resp["error"]; ok {
		if m, ok := errorResp.(map[string]interface{}); ok {
			var status int
			if s, ok := m["status"].(int); ok {
				status = s
			}

			var errType string
			if s, ok := m["type"].(string); ok {
				errType = s
			}

			var reason string
			if s, ok := m["reason"].(string); ok {
				reason = s
			}
			return fmt.Errorf("elasticsearch error: status=%d, type=%s, reason=%s", status, errType, reason)
		}
	}
	return nil
}

func (r *queryRepo) buildElasticQuery(query *entity.Query) map[string]interface{} {
	var queries []map[string]interface{}

	for _, lookup := range query.Lookups {
		field := fmt.Sprintf("tag_%d", lookup.GetTagID())
		var clause map[string]interface{}

		switch lookup.Op {
		case entity.LookupOpEq:
			clause = map[string]interface{}{"term": map[string]interface{}{field: lookup.Val}}
		case entity.LookupOpGt:
			clause = map[string]interface{}{"range": map[string]interface{}{field: map[string]interface{}{"gt": lookup.Val}}}
		case entity.LookupOpLt:
			clause = map[string]interface{}{"range": map[string]interface{}{field: map[string]interface{}{"lt": lookup.Val}}}
		case entity.LookupOpGte:
			clause = map[string]interface{}{"range": map[string]interface{}{field: map[string]interface{}{"gte": lookup.Val}}}
		case entity.LookupOpLte:
			clause = map[string]interface{}{"range": map[string]interface{}{field: map[string]interface{}{"lte": lookup.Val}}}
		case entity.LookupOpIn:
			clause = map[string]interface{}{"terms": map[string]interface{}{field: lookup.Val}}
		}

		if lookup.Not != nil && lookup.GetNot() {
			clause = map[string]interface{}{"bool": map[string]interface{}{"must_not": clause}}
		}
		queries = append(queries, clause)
	}

	// Process Sub-Queries
	for _, subQuery := range query.Queries {
		subClause := r.buildElasticQuery(subQuery)
		if subClause != nil {
			queries = append(queries, subClause)
		}
	}

	// Combine Queries
	if len(queries) == 0 {
		return nil
	}

	var result map[string]interface{}
	if query.Op == entity.QueryOpOr {
		result = map[string]interface{}{"bool": map[string]interface{}{"should": queries}}
	} else {
		result = map[string]interface{}{"bool": map[string]interface{}{"must": queries}}
	}

	if query.Not != nil && query.GetNot() {
		result = map[string]interface{}{"bool": map[string]interface{}{"must_not": result}}
	}

	return result
}

func (r *queryRepo) Close(ctx context.Context) error {
	return r.bulkIndexer.Close(ctx)
}
