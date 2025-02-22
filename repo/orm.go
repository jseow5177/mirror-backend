// Package repo ignore_security_alert_file SQL_INJECTION
package repo

import (
	"cdp/config"
	"cdp/pkg/goutil"
	"context"
	"database/sql"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"reflect"
	"strings"
)

type txKey struct{}

type TxService interface {
	RunTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type BaseRepo interface {
	TxService

	Create(ctx context.Context, model interface{}) error
	CreateMany(ctx context.Context, model interface{}, data interface{}) error
	Get(ctx context.Context, model interface{}, f *Filter) error
	GetMany(ctx context.Context, model interface{}, f *Filter) ([]interface{}, *Pagination, error)
	Count(ctx context.Context, model interface{}, f *Filter) (uint64, error)
	Delete(ctx context.Context, model interface{}, f *Filter) error
	Avg(ctx context.Context, model interface{}, field string, f *Filter) (float64, error)
	GroupBy(ctx context.Context, model, dest interface{}, groupByFields []string, aggregateFields map[string]string, f *Filter) ([]interface{}, error)
	Update(ctx context.Context, model interface{}) error
	Close(ctx context.Context) error
}

type baseRepo struct {
	db *gorm.DB
}

func NewBaseRepo(_ context.Context, mysqlCfg config.MySQL) (BaseRepo, error) {
	db, err := gorm.Open(mysql.Open(mysqlCfg.ToDSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return &baseRepo{
		db: db,
	}, nil
}

func (r *baseRepo) Create(ctx context.Context, data interface{}) error {
	return r.getDb(ctx).Create(data).Error
}

func (r *baseRepo) Count(ctx context.Context, model interface{}, f *Filter) (uint64, error) {
	sqlQuery, args := ToSqlWithArgs(f)

	var count int64
	if err := r.getDb(ctx).Model(model).Where(sqlQuery, args...).Count(&count).Error; err != nil {
		return 0, err
	}
	return uint64(count), nil
}

func (r *baseRepo) Delete(ctx context.Context, model interface{}, f *Filter) error {
	sqlQuery, args := ToSqlWithArgs(f)
	return r.getDb(ctx).Where(sqlQuery, args...).Delete(model).Error
}

func (r *baseRepo) Avg(ctx context.Context, model interface{}, field string, f *Filter) (float64, error) {
	sqlQuery, args := ToSqlWithArgs(f)

	var avg sql.NullFloat64
	if err := r.getDb(ctx).
		Model(model).
		Where(sqlQuery, args...).
		Select(fmt.Sprintf("avg(%s)", field)).
		Scan(&avg).Error; err != nil {
		return 0, err
	}

	if !avg.Valid {
		return 0, nil
	}

	return avg.Float64, nil
}

func (r *baseRepo) GroupBy(ctx context.Context, model, dest interface{}, groupByFields []string, aggregateFields map[string]string, f *Filter) ([]interface{}, error) {
	var (
		selectFields string
		fieldCount   int
	)
	for alias, expr := range aggregateFields {
		selectFields += fmt.Sprintf("%s AS %s", expr, alias)
		fieldCount++

		if fieldCount < len(aggregateFields) {
			selectFields += ", "
		}
	}

	var (
		db = r.getDb(ctx)

		sqlQuery, args = ToSqlWithArgs(f)
	)

	rows, err := db.
		Model(model).
		Where(sqlQuery, args...).
		Select(selectFields).
		Group(strings.Join(groupByFields, ", ")).
		Rows()
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = rows.Close()
	}()

	res := make([]interface{}, 0)
	for rows.Next() {
		if err := db.ScanRows(rows, &dest); err != nil {
			return nil, err
		}
		res = append(res, dest)
	}

	return res, nil
}

func (r *baseRepo) CreateMany(ctx context.Context, model interface{}, data interface{}) error {
	return r.getDb(ctx).Model(model).Create(data).Error
}

func (r *baseRepo) Get(ctx context.Context, model interface{}, f *Filter) error {
	sqlQuery, args := ToSqlWithArgs(f)

	return r.getDb(ctx).Model(model).Where(sqlQuery, args...).First(model).Error
}

func (r *baseRepo) GetMany(ctx context.Context, model interface{}, f *Filter) ([]interface{}, *Pagination, error) {
	var (
		db             = r.getDb(ctx)
		sqlQuery, args = ToSqlWithArgs(f)
		query          = db.Model(model).Where(sqlQuery, args...)
	)

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return nil, nil, err
	}

	pagination := f.Pagination
	if pagination == nil {
		pagination = new(Pagination)
	}

	var (
		limit = pagination.GetLimit()
		page  = pagination.GetPage()
	)
	if page == 0 {
		page = 1
	}

	query = query.Offset(int((page - 1) * limit)).Order("id DESC")
	if limit > 0 {
		query = query.Limit(int(limit + 1))
	}

	var (
		modelElem = reflect.TypeOf(model).Elem()
		queryRes  = reflect.New(reflect.SliceOf(modelElem)).Interface()
	)
	if err := query.Find(queryRes).Error; err != nil {
		return nil, nil, err
	}

	var (
		resElem = reflect.ValueOf(queryRes).Elem()
		res     = make([]interface{}, resElem.Len())
	)
	for i := 0; i < resElem.Len(); i++ {
		res[i] = resElem.Index(i).Addr().Interface() // return addr
	}

	var hasNext bool
	if limit > 0 && len(res) > int(limit) {
		hasNext = true
		res = res[:limit]
	}

	return res, &Pagination{
		Page:    goutil.Uint32(page),
		Limit:   pagination.Limit,
		HasNext: goutil.Bool(hasNext),
		Total:   goutil.Uint32(uint32(count)),
	}, nil
}

func (r *baseRepo) Update(ctx context.Context, model interface{}) error {
	return r.getDb(ctx).Updates(model).Error
}

func (r *baseRepo) RunTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if r.hasTx(ctx) {
		return fn(ctx)
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		ctxWithTx := context.WithValue(ctx, txKey{}, tx)
		if err := fn(ctxWithTx); err != nil {
			return err
		}
		return nil
	})
}

func (r *baseRepo) Close(_ context.Context) error {
	if r.db != nil {
		sqlDB, err := r.db.DB()
		if err != nil {
			return err
		}

		err = sqlDB.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *baseRepo) getDb(ctx context.Context) *gorm.DB {
	db, ok := ctx.Value(txKey{}).(*gorm.DB)
	if !ok {
		db = r.db
	}
	return db
}

func (r *baseRepo) hasTx(ctx context.Context) bool {
	_, ok := ctx.Value(txKey{}).(*gorm.DB)
	return ok
}
