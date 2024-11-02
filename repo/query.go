package repo

import (
	"cdp/config"
	"cdp/entity"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"golang.org/x/sync/errgroup"
	"math"
	"strconv"
	"strings"
	"time"
)

const insertTmpl = "(%v, %v, %v, %v)" // tag_id, mapping_id, ud_id, tag_value

type Query struct{}

type Lookup struct{}

type QueryRepo interface {
	Select(ctx context.Context, query *Query) ([]string, error)
	Insert(ctx context.Context, udTags []*entity.UdTag) error
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

func (r *queryRepo) Insert(ctx context.Context, udTags []*entity.UdTag) error {
	var (
		batch = [][]string{
			make([]string, 0), // str
			make([]string, 0), // int
		}
	)

	for _, udTag := range udTags {
		var (
			tag       = udTag.Tag
			tagValue  = "NULL"
			mappingID = udTag.MappingID
		)

		if tag == nil || mappingID == nil {
			continue
		}

		if udTag.Tag.IsNumeric() {
			if udTag.TagValue != nil {
				if udTag.Tag.IsFloat() {
					f, err := strconv.ParseFloat(*udTag.TagValue, 64)
					if err != nil {
						return err
					}

					// convert to int
					f = math.Round(f * math.Pow(10, 4))
					tagValue = fmt.Sprint(int(f))
				} else {
					tagValue = *udTag.TagValue
				}
			}

			data := fmt.Sprintf(insertTmpl, tag.GetID(), mappingID.GetID(), mappingID.GetUdID(), tagValue)
			batch[1] = append(batch[1], data)
		} else {
			if udTag.TagValue != nil {
				tagValue = fmt.Sprintf("'%s'", *udTag.TagValue)
			}

			data := fmt.Sprintf(insertTmpl, tag.GetID(), mappingID.GetID(), mappingID.GetUdID(), tagValue)
			batch[0] = append(batch[0], data)
		}
	}

	g := new(errgroup.Group)
	for i, data := range batch {
		if len(data) == 0 {
			continue
		}

		tabName := fmt.Sprintf(" %s.cdp_str_tab ", r.database)
		if i == 1 {
			tabName = fmt.Sprintf(" %s.cdp_int_tab ", r.database)
		}

		sql := "INSERT INTO" + tabName + " (`tag_id`, `mapping_id`, `ud_id`, `tag_value`) " + " VALUES " + strings.Join(data, ",")

		g.Go(func() error {
			return r.conn.Exec(ctx, sql)
		})
	}

	return g.Wait()
}

func (r *queryRepo) Select(_ context.Context, _ *Query) ([]string, error) {
	return nil, nil
}

func (r *queryRepo) Close(_ context.Context) error {
	return r.conn.Close()
}
