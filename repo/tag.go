package repo

import (
	"cdp/config"
	"cdp/entity"
	"cdp/pkg/goutil"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Pagination struct {
	Limit *uint32
	Page  *uint32
}

func (p *Pagination) GetLimit() uint32 {
	if p != nil && p.Limit != nil {
		return *p.Limit
	}
	return 0
}

func (p *Pagination) GetPage() uint32 {
	if p != nil && p.Page != nil {
		return *p.Page
	}
	return 0
}

var (
	ErrTagNotFound = errors.New("tag not found")
)

type Tag struct {
	ID         *uint64
	Name       *string
	Desc       *string
	Enum       *string
	ValueType  *uint32
	Status     *uint32
	ExtInfo    *string
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

type TagFilter struct {
	ID         *uint64
	Name       *string
	Desc       *string
	Status     *uint32
	Pagination *Pagination
}

func (f *TagFilter) GetName() string {
	if f != nil && f.Name != nil {
		return *f.Name
	}
	return ""
}

func (f *TagFilter) GetDesc() string {
	if f != nil && f.Desc != nil {
		return *f.Desc
	}
	return ""
}

type TagRepo interface {
	Get(ctx context.Context, f *TagFilter) (*entity.Tag, error)
	GetMany(ctx context.Context, f *TagFilter) ([]*entity.Tag, *entity.Pagination, error)
	Create(ctx context.Context, tag *entity.Tag) (uint64, error)
	Update(ctx context.Context, tag *entity.Tag) error
	Count(ctx context.Context, f *TagFilter) (uint64, error)
	Close(ctx context.Context) error
}

type tagRepo struct {
	orm *gorm.DB
}

func NewTagRepo(_ context.Context, mysqlCfg config.MySQL) (TagRepo, error) {
	orm, err := gorm.Open(mysql.Open(mysqlCfg.ToDSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return &tagRepo{orm: orm}, nil
}

func (r *tagRepo) Count(_ context.Context, _ *TagFilter) (uint64, error) {
	var count int64
	if err := r.orm.Model(&Tag{}).Where("status != ?", entity.TagStatusDeleted).Count(&count).Error; err != nil {
		return 0, err
	}
	return uint64(count), nil
}

func (r *tagRepo) Get(_ context.Context, f *TagFilter) (*entity.Tag, error) {
	tag := new(Tag)
	if err := r.orm.Where(f).First(tag).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}
	return ToTag(tag)
}

func (r *tagRepo) GetMany(_ context.Context, f *TagFilter) ([]*entity.Tag, *entity.Pagination, error) {
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
		cond += "LOWER(\"desc\") LIKE ?"
		args = append(args, fmt.Sprintf("%%%s%%", f.GetDesc()))
	}

	if cond != "" {
		cond += " AND "
	}
	cond += "status != ?"
	args = append(args, entity.TagStatusDeleted)

	var count int64
	if err := r.orm.Model(&Tag{}).Where(cond, args...).Count(&count).Error; err != nil {
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
		offset = (page - 1) * limit
		mTags  = make([]*Tag, 0)
	)
	query := r.orm.Where(cond, args...).Offset(int(offset))
	if limit > 0 {
		query = query.Limit(int(limit + 1))
	}

	if err := query.Find(&mTags).Error; err != nil {
		return nil, nil, err
	}

	var hasNext bool
	if limit > 0 && len(mTags) > int(limit) {
		hasNext = true
		mTags = mTags[:limit]
	}

	tags := make([]*entity.Tag, len(mTags))
	for i, mTag := range mTags {
		tag, err := ToTag(mTag)
		if err != nil {
			return nil, nil, err
		}
		tags[i] = tag
	}

	return tags, &entity.Pagination{
		Page:    goutil.Uint32(page),
		Limit:   f.Pagination.Limit, // may be nil
		HasNext: goutil.Bool(hasNext),
		Total:   goutil.Int64(count),
	}, nil
}

func (r *tagRepo) Create(_ context.Context, tag *entity.Tag) (uint64, error) {
	tagModel, err := ToTagModel(tag)
	if err != nil {
		return 0, err
	}

	if err := r.orm.Create(&tagModel).Error; err != nil {
		return 0, err
	}

	return tagModel.GetID(), nil
}

func (r *tagRepo) Update(_ context.Context, _ *entity.Tag) error {
	panic("implement me")
}

func (r *tagRepo) Close(_ context.Context) error {
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
		Desc:       tag.Desc,
		ValueType:  tag.ValueType,
		Status:     tag.Status,
		ExtInfo:    goutil.String(extInfo),
		Enum:       goutil.String(string(enum)),
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
		Desc:       tag.Desc,
		Status:     tag.Status,
		ValueType:  tag.ValueType,
		Enum:       enum,
		ExtInfo:    extInfo,
		CreateTime: tag.CreateTime,
		UpdateTime: tag.UpdateTime,
	}, nil
}
