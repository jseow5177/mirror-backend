// ignore_security_alert_file SQL_INJECTION
package repo

import (
	"cdp/config"
	"cdp/pkg/goutil"
	"context"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"reflect"
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

func (r *baseRepo) BuildConditions(baseConditions, extraConditions []*Condition) []*Condition {
	if len(baseConditions) == 0 {
		return extraConditions
	} else {
		if len(extraConditions) == 0 {
			baseConditions[len(baseConditions)-1].NextLogicalOp = ""
		} else {
			baseConditions[len(baseConditions)-1].NextLogicalOp = LogicalOpAnd
		}
		return append(baseConditions, extraConditions...)
	}
}

func (r *baseRepo) Create(ctx context.Context, data interface{}) error {
	return r.getDb(ctx).Create(data).Error
}

func (r *baseRepo) Count(ctx context.Context, model interface{}, f *Filter) (uint64, error) {
	sql, args := ToSqlWithArgs(f)

	var count int64
	if err := r.getDb(ctx).Model(model).Where(sql, args...).Count(&count).Error; err != nil {
		return 0, err
	}
	return uint64(count), nil
}

func (r *baseRepo) CreateMany(ctx context.Context, model interface{}, data interface{}) error {
	return r.getDb(ctx).Model(model).Create(data).Error
}

func (r *baseRepo) Get(ctx context.Context, model interface{}, f *Filter) error {
	sql, args := ToSqlWithArgs(f)

	return r.getDb(ctx).Model(model).Where(sql, args...).First(model).Error
}

func (r *baseRepo) GetMany(ctx context.Context, model interface{}, f *Filter) ([]interface{}, *Pagination, error) {
	var (
		db        = r.getDb(ctx)
		sql, args = ToSqlWithArgs(f)
		query     = db.Model(model).Where(sql, args...)
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

	query = query.Offset(int((page - 1) * limit))
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
