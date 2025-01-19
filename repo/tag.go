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
	ErrTagNotFound = errutil.NotFoundError(errors.New("tag not found"))
)

type Tag struct {
	ID         *uint64
	Name       *string
	TagDesc    *string
	Enum       *string
	ValueType  *uint32
	Status     *uint32
	ExtInfo    *string
	CreatorID  *uint64
	TenantID   *uint64
	CreateTime *uint64
	UpdateTime *uint64
}

func (m *Tag) TableName() string {
	return "tag_tab"
}

func (m *Tag) GetID() uint64 {
	if m != nil && m.ID != nil {
		return *m.ID
	}
	return 0
}

func (m *Tag) GetStatus() uint32 {
	if m != nil && m.Status != nil {
		return *m.Status
	}
	return 0
}

func (m *Tag) GetValueType() uint32 {
	if m != nil && m.ValueType != nil {
		return *m.ValueType
	}
	return 0
}

func (m *Tag) GetEnum() string {
	if m != nil && m.Enum != nil {
		return *m.Enum
	}
	return ""
}

func (m *Tag) GetExtInfo() string {
	if m != nil && m.ExtInfo != nil {
		return *m.ExtInfo
	}
	return ""
}

type TagRepo interface {
	Create(ctx context.Context, tag *entity.Tag) (uint64, error)
	GetByID(ctx context.Context, tenantID, tagID uint64) (*entity.Tag, error)
	CountByTenantID(ctx context.Context, tenantID uint64) (uint64, error)
	GetByKeyword(ctx context.Context, tenantID uint64, keyword string, p *Pagination) ([]*entity.Tag, *Pagination, error)
	GetByName(ctx context.Context, tenantID uint64, name string) (*entity.Tag, error)
}

type tagRepo struct {
	baseRepo BaseRepo
}

func NewTagRepo(_ context.Context, baseRepo BaseRepo) TagRepo {
	return &tagRepo{baseRepo: baseRepo}
}

func (r *tagRepo) CountByTenantID(ctx context.Context, tenantID uint64) (uint64, error) {
	return r.count(ctx, tenantID, nil, true)
}

func (r *tagRepo) count(ctx context.Context, tenantID uint64, conditions []*Condition, filterDelete bool) (uint64, error) {
	return r.baseRepo.Count(ctx, new(Tag), &Filter{
		Conditions: append(r.getBaseConditions(tenantID), r.mayAddDeleteFilter(conditions, filterDelete)...),
	})
}

func (r *tagRepo) mayAddDeleteFilter(conditions []*Condition, filterDelete bool) []*Condition {
	if filterDelete {
		return append(conditions, &Condition{
			Field: "status",
			Value: entity.TagStatusDeleted,
			Op:    OpNotEq,
		})

	}
	return conditions
}

func (r *tagRepo) GetByName(ctx context.Context, tenantID uint64, name string) (*entity.Tag, error) {
	return r.get(ctx, tenantID, []*Condition{
		{
			Field: "name",
			Value: name,
			Op:    OpEq,
		},
	}, true)
}

func (r *tagRepo) GetByID(ctx context.Context, tenantID, tagID uint64) (*entity.Tag, error) {
	return r.get(ctx, tenantID, []*Condition{
		{
			Field: "tag_id",
			Value: tagID,
			Op:    OpEq,
		},
	}, true)
}

func (r *tagRepo) GetByKeyword(ctx context.Context, tenantID uint64, keyword string, p *Pagination) ([]*entity.Tag, *Pagination, error) {
	return r.getMany(ctx, tenantID, []*Condition{
		{
			Field:         "LOWER(name)",
			Value:         fmt.Sprintf("%%%s%%", keyword),
			Op:            OpLike,
			NextLogicalOp: LogicalOpOr,
			OpenBracket:   true,
		},
		{
			Field:        "LOWER(tag_desc)",
			Value:        fmt.Sprintf("%%%s%%", keyword),
			Op:           OpLike,
			CloseBracket: true,
		},
	}, true, p)
}

func (r *tagRepo) getMany(ctx context.Context, tenantID uint64, conditions []*Condition, filterDelete bool, p *Pagination) ([]*entity.Tag, *Pagination, error) {
	res, pNew, err := r.baseRepo.GetMany(ctx, new(Tag), &Filter{
		Conditions: append(r.getBaseConditions(tenantID), r.mayAddDeleteFilter(conditions, filterDelete)...),
		Pagination: p,
	})
	if err != nil {
		return nil, nil, err
	}

	tags := make([]*entity.Tag, 0, len(res))
	for _, m := range res {
		tag, err := ToTag(m.(*Tag))
		if err != nil {
			return nil, nil, err
		}
		tags = append(tags, tag)
	}

	return tags, pNew, nil
}

func (r *tagRepo) get(ctx context.Context, tenantID uint64, conditions []*Condition, filterDelete bool) (*entity.Tag, error) {
	tag := new(Tag)

	if err := r.baseRepo.Get(ctx, tag, &Filter{
		Conditions: append(r.getBaseConditions(tenantID), r.mayAddDeleteFilter(conditions, filterDelete)...),
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}

	return ToTag(tag)
}

func (r *tagRepo) getBaseConditions(tenantID uint64) []*Condition {
	return []*Condition{
		{
			Field:         "tenant_id",
			Value:         tenantID,
			Op:            OpEq,
			NextLogicalOp: LogicalOpAnd,
		},
	}
}

func (r *tagRepo) Create(ctx context.Context, tag *entity.Tag) (uint64, error) {
	tagModel, err := ToTagModel(tag)
	if err != nil {
		return 0, err
	}

	if err := r.baseRepo.Create(ctx, tagModel); err != nil {
		return 0, err
	}

	return tagModel.GetID(), nil
}

func ToTagModel(tag *entity.Tag) (*Tag, error) {
	extInfo, err := tag.GetExtInfo().ToString()
	if err != nil {
		return nil, err
	}

	enum, err := json.Marshal(tag.GetEnum())
	if err != nil {
		return nil, err
	}

	return &Tag{
		ID:         tag.ID,
		Name:       tag.Name,
		TagDesc:    tag.TagDesc,
		ValueType:  goutil.Uint32(uint32(tag.GetValueType())),
		Status:     goutil.Uint32(uint32(tag.GetStatus())),
		ExtInfo:    goutil.String(extInfo),
		Enum:       goutil.String(string(enum)),
		CreatorID:  tag.CreatorID,
		TenantID:   tag.TenantID,
		CreateTime: tag.CreateTime,
		UpdateTime: tag.UpdateTime,
	}, nil
}

func ToTag(tag *Tag) (*entity.Tag, error) {
	extInfo := new(entity.TagExtInfo)
	if err := json.Unmarshal([]byte(tag.GetExtInfo()), extInfo); err != nil {
		return nil, err
	}

	enum := make([]string, 0)
	if err := json.Unmarshal([]byte(tag.GetEnum()), &enum); err != nil {
		return nil, err
	}

	return &entity.Tag{
		ID:         tag.ID,
		Name:       tag.Name,
		TagDesc:    tag.TagDesc,
		Status:     entity.TagStatus(tag.GetStatus()),
		ValueType:  entity.TagValueType(tag.GetValueType()),
		Enum:       enum,
		ExtInfo:    extInfo,
		CreatorID:  tag.CreatorID,
		TenantID:   tag.TenantID,
		CreateTime: tag.CreateTime,
		UpdateTime: tag.UpdateTime,
	}, nil
}
