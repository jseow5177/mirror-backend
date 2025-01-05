package repo

import (
	"cdp/config"
	"context"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
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
	Update(ctx context.Context, model interface{}) error
	BuildConditions(baseConditions, extraConditions []*Condition) []*Condition
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

func (r *baseRepo) CreateMany(ctx context.Context, model interface{}, data interface{}) error {
	return r.getDb(ctx).Model(model).Create(data).Error
}

func (r *baseRepo) Get(ctx context.Context, model interface{}, f *Filter) error {
	sql, args := ToSqlWithArgs(f)

	return r.getDb(ctx).Model(model).Where(sql, args...).First(model).Error
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
