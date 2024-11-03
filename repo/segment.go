package repo

import (
	"cdp/config"
	"cdp/entity"
	"cdp/pkg/goutil"
	"context"
	"encoding/json"
	"errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	ErrSegmentNotFound = errors.New("segment not found")
)

type Segment struct {
	ID         *uint64
	Name       *string
	Desc       *string
	Query      *string
	Status     *uint32
	CreateTime *uint64
	UpdateTime *uint64
}

func (m *Segment) TableName() string {
	return "segment_tab"
}

func (m *Segment) GetID() uint64 {
	if m != nil && m.ID != nil {
		return *m.ID
	}
	return 0
}

func (m *Segment) GetQuery() string {
	if m != nil && m.Query != nil {
		return *m.Query
	}
	return ""
}

type SegmentFilter struct {
	ID     *uint64
	Name   *string
	Status *uint32
}

type SegmentRepo interface {
	Get(ctx context.Context, f *SegmentFilter) (*entity.Segment, error)
	Create(ctx context.Context, segment *entity.Segment) (uint64, error)
	Close(ctx context.Context) error
}

type segmentRepo struct {
	orm *gorm.DB
}

func NewSegmentRepo(_ context.Context, mysqlCfg config.MySQL) (SegmentRepo, error) {
	orm, err := gorm.Open(mysql.Open(mysqlCfg.ToDSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return &segmentRepo{orm: orm}, nil
}

func (r *segmentRepo) Get(_ context.Context, f *SegmentFilter) (*entity.Segment, error) {
	segment := new(Segment)
	if err := r.orm.Where(f).First(segment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSegmentNotFound
		}
		return nil, err
	}
	return ToSegment(segment)
}

func (r *segmentRepo) Create(_ context.Context, segment *entity.Segment) (uint64, error) {
	segmentModel, err := ToSegmentModel(segment)
	if err != nil {
		return 0, err
	}

	if err := r.orm.Create(&segmentModel).Error; err != nil {
		return 0, err
	}

	return segmentModel.GetID(), nil
}

func (r *segmentRepo) Close(_ context.Context) error {
	if r.orm != nil {
		sqlDB, err := r.orm.DB()
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

func ToSegmentModel(segment *entity.Segment) (*Segment, error) {
	query, err := segment.GetQuery().ToString()
	if err != nil {
		return nil, err
	}

	return &Segment{
		ID:         segment.ID,
		Name:       segment.Name,
		Desc:       segment.Desc,
		Status:     segment.Status,
		Query:      goutil.String(query),
		CreateTime: segment.CreateTime,
		UpdateTime: segment.UpdateTime,
	}, nil
}

func ToSegment(segment *Segment) (*entity.Segment, error) {
	query := new(entity.Query)
	if err := json.Unmarshal([]byte(segment.GetQuery()), query); err != nil {
		return nil, err
	}

	return &entity.Segment{
		ID:         segment.ID,
		Name:       segment.Name,
		Desc:       segment.Desc,
		Query:      query,
		Status:     segment.Status,
		CreateTime: segment.CreateTime,
		UpdateTime: segment.UpdateTime,
	}, nil
}
