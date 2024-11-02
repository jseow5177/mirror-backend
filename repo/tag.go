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
	ID     *uint64
	Name   *string
	Status *uint32
}

type TagRepo interface {
	Get(ctx context.Context, f *TagFilter) (*entity.Tag, error)
	GetMany(ctx context.Context, f *TagFilter) ([]*entity.Tag, error)
	Create(ctx context.Context, tag *entity.Tag) (uint64, error)
	Update(ctx context.Context, tag *entity.Tag) error
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

func (r *tagRepo) GetMany(_ context.Context, _ *TagFilter) ([]*entity.Tag, error) {
	panic("implement me")
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
		ValueType:  goutil.Uint32(tag.GetValueType()),
		Status:     goutil.Uint32(tag.GetStatus()),
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
