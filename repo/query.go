package repo

import (
	"cdp/config"
	"cdp/entity"
	"context"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"github.com/rs/zerolog/log"
	"net/http"
	"strings"
	"time"
)

type QueryRepo interface {
	CreateStore(_ context.Context, tenantName string) error
	BatchUpsert(ctx context.Context, tenantName string, udTagVals []*entity.UdTagVal) error
	OnInsertSuccess() chan struct{}
	OnInsertFailure() chan struct{}
	Close(ctx context.Context) error
}

type queryRepo struct {
	client          *elasticsearch.Client
	bulkIndexer     esutil.BulkIndexer
	onInsertSuccess chan struct{}
	onInsertFailure chan struct{}
}

var (
	defaultNumWorkers           = 10
	defaultFlushBytes           = 1_000_000
	defaultFlushIntervalSeconds = 5
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
		onInsertSuccess: make(chan struct{}, 100_000),
		onInsertFailure: make(chan struct{}, 100_000),
	}, nil
}

func (r *queryRepo) CreateStore(_ context.Context, tenantName string) error {
	_, err := r.client.Indices.Create(tenantName)
	if err != nil {
		return err
	}
	return nil
}

func (r *queryRepo) OnInsertSuccess() chan struct{} {
	return r.onInsertSuccess
}

func (r *queryRepo) OnInsertFailure() chan struct{} {
	return r.onInsertFailure
}

// Bulk indexer: https://github.com/elastic/go-elasticsearch/blob/main/_examples/bulk/indexer.go
func (r *queryRepo) BatchUpsert(ctx context.Context, tenantName string, udTagVals []*entity.UdTagVal) error {
	for _, udTagVal := range udTagVals {
		docID := udTagVal.ToDocID()

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
				select {
				case r.onInsertFailure <- struct{}{}:
				default:
				}

				if err != nil {
					log.Ctx(ctx).Error().Msgf("fail to index doc: %v, docID: %v, data: %v",
						err, docID, data)
				} else {
					log.Ctx(ctx).Error().Msgf("fail to index doc: %s: %s, docID: %v, data: %v",
						res.Error.Type, res.Error.Reason, docID, data)
				}
			},
		}); err != nil {
			log.Ctx(ctx).Error().Msgf("fail to add udTagVal to indexer: %v, docID: %v, data: %v", err, docID, data)
			return err
		}
	}

	return nil
}

func (r *queryRepo) Close(ctx context.Context) error {
	return r.bulkIndexer.Close(ctx)
}
