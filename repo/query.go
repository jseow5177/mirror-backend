package repo

import (
	"cdp/config"
	"cdp/entity"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"strings"
	"time"
)

const insertTmpl = "(%v, %v, '%v', %v)" // tag_id, mapping_id, ud_id, tag_value

type Query struct{}

type Lookup struct{}

type QueryRepo interface {
	Select(ctx context.Context, query *Query) ([]string, error)
	Insert(ctx context.Context, udTags *entity.UdTags) error
	Close(ctx context.Context) error
}

type queryRepo struct {
	database string
	conn     driver.Conn
}

func NewQueryRepo(ctx context.Context, ckCfg config.ClickHouse) (QueryRepo, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr:  ckCfg.Addr,
		Debug: ckCfg.Debug,
		Auth: clickhouse.Auth{
			Database: ckCfg.Database,
			Username: ckCfg.Username,
			Password: ckCfg.Password,
		},
		TLS:             &tls.Config{},
		DialTimeout:     time.Duration(ckCfg.DialTimeoutSeconds) * time.Second,
		ConnMaxLifetime: time.Duration(ckCfg.ConnMaxLifetimeSeconds) * time.Second,
		MaxIdleConns:    ckCfg.MaxIdleConns,
		MaxOpenConns:    ckCfg.MaxOpenConns,
	})
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		return nil, err
	}

	return &queryRepo{
		database: ckCfg.Database,
		conn:     conn,
	}, nil
}

func (r *queryRepo) Insert(ctx context.Context, udTags *entity.UdTags) error {
	var (
		batch = make([]string, 0, len(udTags.Data))
		tag   = udTags.Tag
	)
	if tag == nil {
		return nil
	}

	for _, udTag := range udTags.Data {
		var (
			tagValue  = "NULL"
			mappingID = udTag.MappingID
		)

		if mappingID == nil {
			continue
		}

		if udTag.TagValue != nil {
			if tag.IsNumeric() {
				tagValue = fmt.Sprintf("'%s'", *udTag.TagValue)
			} else {
				tagValue = fmt.Sprintf("%s", *udTag.TagValue)
			}
		}

		data := fmt.Sprintf(insertTmpl, tag.GetID(), mappingID.GetID(), mappingID.GetUdID(), tagValue)
		batch = append(batch, data)
	}

	if len(batch) == 0 {
		return nil
	}

	tabName := tag.GetCkTagName()
	sql := "INSERT INTO" + fmt.Sprintf(" %s ", tabName) + "(`tag_id`, `mapping_id`, `ud_id`, `tag_value`) " + " VALUES " + strings.Join(batch, ",")

	return r.conn.Exec(ctx, sql)
}

func (r *queryRepo) Select(_ context.Context, _ *Query) ([]string, error) {
	return nil, nil
}

func (r *queryRepo) Close(_ context.Context) error {
	return r.conn.Close()
}
