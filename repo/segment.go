package repo

import (
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
)

var (
	ErrSegmentNotFound = errutil.NotFoundError(errors.New("segment not found"))
)

type Segment struct {
	ID          *uint64
	Name        *string
	SegmentDesc *string
	Criteria    *string
	Status      *uint32
	CreatorID   *uint64
	TenantID    *uint64
	CreateTime  *uint64
	UpdateTime  *uint64
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

type SegmentRepo interface {
	Create(ctx context.Context, segment *entity.Segment) (uint64, error)
	GetByID(ctx context.Context, tenantID, segmentID uint64) (*entity.Segment, error)
	GetByName(ctx context.Context, tenantID uint64, name string) (*entity.Segment, error)
	GetManyByKeyword(ctx context.Context, tenantID uint64, keyword string, p *Pagination) ([]*entity.Segment, *Pagination, error)
	CountByTenantID(ctx context.Context, tenantID uint64) (uint64, error)
}

type segmentRepo struct {
	baseRepo BaseRepo
}

func NewSegmentRepo(_ context.Context, baseRepo BaseRepo) SegmentRepo {
	return &segmentRepo{baseRepo: baseRepo}
}

func (r *segmentRepo) CountByTenantID(ctx context.Context, tenantID uint64) (uint64, error) {
	return r.count(ctx, tenantID, nil, true)
}

func (r *segmentRepo) count(ctx context.Context, tenantID uint64, conditions []*Condition, filterDelete bool) (uint64, error) {
	return r.baseRepo.Count(ctx, new(Segment), &Filter{
		Conditions: append(r.getBaseConditions(tenantID), r.mayAddDeleteFilter(conditions, filterDelete)...),
	})
}

func (r *segmentRepo) GetByName(ctx context.Context, tenantID uint64, name string) (*entity.Segment, error) {
	return r.get(ctx, tenantID, []*Condition{
		{
			Field: "name",
			Value: name,
			Op:    OpEq,
		},
	}, true)
}

func (r *segmentRepo) GetByID(ctx context.Context, tenantID, segmentID uint64) (*entity.Segment, error) {
	return r.get(ctx, tenantID, []*Condition{
		{
			Field: "id",
			Value: segmentID,
			Op:    OpEq,
		},
	}, true)
}

func (r *segmentRepo) GetManyByKeyword(ctx context.Context, tenantID uint64, keyword string, p *Pagination) ([]*entity.Segment, *Pagination, error) {
	return r.getMany(ctx, tenantID, []*Condition{
		{
			Field:         "LOWER(name)",
			Value:         fmt.Sprintf("%%%s%%", keyword),
			Op:            OpLike,
			NextLogicalOp: LogicalOpOr,
			OpenBracket:   true,
		},
		{
			Field:        "LOWER(segment_desc)",
			Value:        fmt.Sprintf("%%%s%%", keyword),
			Op:           OpLike,
			CloseBracket: true,
		},
	}, true, p)
}

func (r *segmentRepo) getMany(ctx context.Context, tenantID uint64, conditions []*Condition, filterDelete bool, p *Pagination) ([]*entity.Segment, *Pagination, error) {
	res, pNew, err := r.baseRepo.GetMany(ctx, new(Segment), &Filter{
		Conditions: append(r.getBaseConditions(tenantID), r.mayAddDeleteFilter(conditions, filterDelete)...),
		Pagination: p,
	})
	if err != nil {
		return nil, nil, err
	}

	segments := make([]*entity.Segment, 0, len(res))
	for _, m := range res {
		segment, err := ToSegment(m.(*Segment))
		if err != nil {
			return nil, nil, err
		}
		segments = append(segments, segment)
	}

	return segments, pNew, nil
}

func (r *segmentRepo) get(ctx context.Context, tenantID uint64, conditions []*Condition, filterDelete bool) (*entity.Segment, error) {
	segment := new(Segment)

	if err := r.baseRepo.Get(ctx, segment, &Filter{
		Conditions: append(r.getBaseConditions(tenantID), r.mayAddDeleteFilter(conditions, filterDelete)...),
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSegmentNotFound
		}
		return nil, err
	}

	return ToSegment(segment)
}

func (r *segmentRepo) mayAddDeleteFilter(conditions []*Condition, filterDelete bool) []*Condition {
	if filterDelete {
		return append(conditions, &Condition{
			Field: "status",
			Value: entity.SegmentStatusDeleted,
			Op:    OpNotEq,
		})

	}
	return conditions
}

func (r *segmentRepo) getBaseConditions(tenantID uint64) []*Condition {
	return []*Condition{
		{
			Field:         "tenant_id",
			Value:         tenantID,
			Op:            OpEq,
			NextLogicalOp: LogicalOpAnd,
		},
	}
}

func (r *segmentRepo) Create(ctx context.Context, segment *entity.Segment) (uint64, error) {
	segmentModel, err := ToSegmentModel(segment)
	if err != nil {
		return 0, err
	}

	if err := r.baseRepo.Create(ctx, segmentModel); err != nil {
		return 0, err
	}

	return segmentModel.GetID(), nil
}

func ToSegmentModel(segment *entity.Segment) (*Segment, error) {
	query, err := segment.GetCriteria().ToString()
	if err != nil {
		return nil, err
	}

	return &Segment{
		ID:          segment.ID,
		Name:        segment.Name,
		SegmentDesc: segment.SegmentDesc,
		Status:      goutil.Uint32(uint32(segment.GetStatus())),
		Criteria:    goutil.String(query),
		TenantID:    segment.TenantID,
		CreatorID:   segment.CreatorID,
		CreateTime:  segment.CreateTime,
		UpdateTime:  segment.UpdateTime,
	}, nil
}

func ToSegment(segment *Segment) (*entity.Segment, error) {
	query := new(entity.Query)
	if err := json.Unmarshal([]byte(segment.GetCriteria()), query); err != nil {
		return nil, err
	}

	return &entity.Segment{
		ID:          segment.ID,
		Name:        segment.Name,
		SegmentDesc: segment.SegmentDesc,
		Criteria:    query,
		Status:      entity.SegmentStatus(segment.GetStatus()),
		TenantID:    segment.TenantID,
		CreatorID:   segment.CreatorID,
		CreateTime:  segment.CreateTime,
		UpdateTime:  segment.UpdateTime,
	}, nil
}
