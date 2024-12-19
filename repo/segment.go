package repo

import (
	"cdp/config"
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	ErrSegmentNotFound = errutil.NotFoundError(errors.New("segment not found"))
)

type Segment struct {
	ID         *uint64
	Name       *string
	Desc       *string
	Criteria   *string
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

func (m *Segment) GetCriteria() string {
	if m != nil && m.Criteria != nil {
		return *m.Criteria
	}
	return ""
}

func (m *Segment) GetStatus() uint32 {
	if m != nil && m.Status != nil {
		return *m.Status
	}
	return 0
}

type SegmentFilter struct {
	ID         *uint64
	Name       *string
	Desc       *string
	Status     *uint32
	Pagination *Pagination `gorm:"-"`
}

func (f *SegmentFilter) GetName() string {
	if f != nil && f.Name != nil {
		return *f.Name
	}
	return ""
}

func (f *SegmentFilter) GetDesc() string {
	if f != nil && f.Desc != nil {
		return *f.Desc
	}
	return ""
}

type SegmentRepo interface {
	Get(ctx context.Context, f *SegmentFilter) (*entity.Segment, error)
	GetMany(ctx context.Context, f *SegmentFilter) ([]*entity.Segment, *entity.Pagination, error)
	Create(ctx context.Context, segment *entity.Segment) (uint64, error)
	Count(ctx context.Context, f *SegmentFilter) (uint64, error)
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

func (r *segmentRepo) Count(_ context.Context, _ *SegmentFilter) (uint64, error) {
	var count int64
	if err := r.orm.Model(&Segment{}).Where("status != ?", entity.SegmentStatusDeleted).Count(&count).Error; err != nil {
		return 0, err
	}
	return uint64(count), nil
}

func (r *segmentRepo) GetMany(_ context.Context, f *SegmentFilter) ([]*entity.Segment, *entity.Pagination, error) {
	var (
		cond string
		args = make([]interface{}, 0)
	)
	if f.Name != nil {
		cond += "LOWER(name) LIKE ?"
		args = append(args, fmt.Sprintf("%%%s%%", f.GetName()))
	}

	if f.Desc != nil {
		if cond != "" {
			cond += " OR "
		}
		cond += "LOWER(`desc`) LIKE ?"
		args = append(args, fmt.Sprintf("%%%s%%", f.GetDesc()))
	}

	if cond != "" {
		cond += " AND "
	}
	cond += "status != ?"
	args = append(args, entity.SegmentStatusDeleted)

	var count int64
	if err := r.orm.Model(new(Segment)).Where(cond, args...).Count(&count).Error; err != nil {
		return nil, nil, err
	}

	var (
		limit = f.Pagination.GetLimit()
		page  = f.Pagination.GetPage()
	)
	if page == 0 {
		page = 1
	}

	var (
		offset    = (page - 1) * limit
		mSegments = make([]*Segment, 0)
	)
	query := r.orm.Where(cond, args...).Offset(int(offset))
	if limit > 0 {
		query = query.Limit(int(limit + 1))
	}

	if err := query.Find(&mSegments).Error; err != nil {
		return nil, nil, err
	}

	var hasNext bool
	if limit > 0 && len(mSegments) > int(limit) {
		hasNext = true
		mSegments = mSegments[:limit]
	}

	segments := make([]*entity.Segment, len(mSegments))
	for i, mSegment := range mSegments {
		segment, err := ToSegment(mSegment)
		if err != nil {
			return nil, nil, err
		}
		segments[i] = segment
	}

	return segments, &entity.Pagination{
		Page:    goutil.Uint32(page),
		Limit:   f.Pagination.Limit, // may be nil
		HasNext: goutil.Bool(hasNext),
		Total:   goutil.Int64(count),
	}, nil
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
	query, err := segment.GetCriteria().ToString()
	if err != nil {
		return nil, err
	}

	return &Segment{
		ID:         segment.ID,
		Name:       segment.Name,
		Desc:       segment.Desc,
		Status:     goutil.Uint32(uint32(segment.GetStatus())),
		Criteria:   goutil.String(query),
		CreateTime: segment.CreateTime,
		UpdateTime: segment.UpdateTime,
	}, nil
}

func ToSegment(segment *Segment) (*entity.Segment, error) {
	query := new(entity.Query)
	if err := json.Unmarshal([]byte(segment.GetCriteria()), query); err != nil {
		return nil, err
	}

	return &entity.Segment{
		ID:         segment.ID,
		Name:       segment.Name,
		Desc:       segment.Desc,
		Criteria:   query,
		Status:     entity.SegmentStatus(segment.GetStatus()),
		CreateTime: segment.CreateTime,
		UpdateTime: segment.UpdateTime,
	}, nil
}
