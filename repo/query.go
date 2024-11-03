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

type QueryRepo interface {
	Select(ctx context.Context) ([]string, error)
	Count(ctx context.Context, query *entity.Query) (uint64, error)
	Insert(ctx context.Context, udTags *entity.UdTags) error
	Close(ctx context.Context) error
}

type queryRepo struct {
	database string
	conn     driver.Conn
	tagRepo  TagRepo
}

func NewQueryRepo(ctx context.Context, ckCfg config.ClickHouse, tagRepo TagRepo) (QueryRepo, error) {
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
		tagRepo:  tagRepo,
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

	tabName := tag.GetCkTabName()
	sql := "INSERT INTO" + fmt.Sprintf(" %s ", tabName) +
		"(`tag_id`, `mapping_id`, `ud_id`, `tag_value`) " + " VALUES " + strings.Join(batch, ",")

	return r.conn.Exec(ctx, sql)
}

func (r *queryRepo) Select(_ context.Context) ([]string, error) {
	return nil, nil
}

func (r *queryRepo) Count(ctx context.Context, query *entity.Query) (uint64, error) {
	cond, err := r.constructSQL(ctx, query)
	if err != nil {
		return 0, err
	}

	var count uint64
	sql := fmt.Sprintf("SELECT bitmapCardinality ( (%s) )", cond)

	if err := r.conn.QueryRow(ctx, sql).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (r *queryRepo) constructSQL(ctx context.Context, query *entity.Query) (string, error) {
	var (
		stmt string
		sqls = make([]string, 0)
		tmpl = "bitmapAnd( (%s), (%s) )"
	)
	if query.GetOp() == entity.QueryOpOr {
		tmpl = "bitmapOr( (%s), (%s) )"
	}

	for _, query := range query.Queries {
		sql, err := r.constructSQL(ctx, query)
		if err != nil {
			return "", err
		}
		sqls = append(sqls, sql)
	}

	for _, lookup := range query.Lookups {
		lookupTmpl := `
			SELECT groupBitmapState(mapping_id) from %s final
			where tag_id = %d AND %s
		`

		tag, err := r.tagRepo.Get(context.Background(), &TagFilter{
			ID: lookup.TagID,
		})
		if err != nil {
			return "", err
		}

		var cond string
		if lookup.Eq != nil {
			cond += fmt.Sprintf("tag_value = '%s'", lookup.GetEq())
		} else if len(lookup.In) > 0 {
			ins := make([]string, 0, len(lookup.In))
			for _, in := range lookup.In {
				ins = append(ins, fmt.Sprintf("'%s'", in))
			}
			cond += fmt.Sprintf("tag_value IN (%s)", strings.Join(ins, ","))
		} else if lookup.Range != nil {
			if lookup.Range.Gte != nil {
				cond = fmt.Sprintf("tag_value >= '%s'", lookup.Range.GetGte())
			} else if lookup.Range.Gt != nil {
				cond = fmt.Sprintf("tag_value > '%s'", lookup.Range.GetGt())
			}

			if lookup.Range.Lte != nil {
				if cond != "" {
					cond += " AND "
				}
				cond += fmt.Sprintf("tag_value <= '%s'", lookup.Range.GetLte())
			} else if lookup.Range.Lt != nil {
				if cond != "" {
					cond += " AND "
				}
				cond += fmt.Sprintf("tag_value < '%s'", lookup.Range.GetLt())
			}
		}

		sqls = append(sqls, fmt.Sprintf(lookupTmpl, tag.GetCkTabName(), lookup.GetTagID(), cond))
	}

	if len(sqls) > 0 {
		stmt = sqls[0]

		for i := 1; i < len(sqls); i++ {
			stmt = fmt.Sprintf(tmpl, stmt, sqls[i])
		}
	}

	return stmt, nil
}

func (r *queryRepo) Close(_ context.Context) error {
	return r.conn.Close()
}
