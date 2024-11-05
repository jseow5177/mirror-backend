package handler

import (
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"cdp/pkg/validator"
	"cdp/repo"
	"context"
	"errors"
	"github.com/rs/zerolog/log"
	"time"
)

type TagHandler interface {
	CreateTag(ctx context.Context, req *CreateTagRequest, res *CreateTagResponse) error
	GetTags(ctx context.Context, req *GetTagsRequest, res *GetTagsResponse) error
	CountTags(ctx context.Context, req *CountTagsRequest, res *CountTagsResponse) error
}

type tagHandler struct {
	tagRepo repo.TagRepo
}

func NewTagHandler(tagRepo repo.TagRepo) TagHandler {
	return &tagHandler{
		tagRepo: tagRepo,
	}
}

type CountTagsRequest struct{}

type CountTagsResponse struct {
	Count *uint64 `json:"count,omitempty"`
}

var CountTagsValidator = validator.MustForm(map[string]validator.Validator{})

func (h *tagHandler) CountTags(ctx context.Context, req *CountTagsRequest, res *CountTagsResponse) error {
	if err := CountTagsValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	count, err := h.tagRepo.Count(ctx, nil)
	if err != nil {
		return err
	}

	res.Count = goutil.Uint64(count)

	return nil
}

type GetTagsRequest struct {
	Name       *string            `json:"name,omitempty"`
	Desc       *string            `json:"desc,omitempty"`
	Pagination *entity.Pagination `json:"pagination,omitempty"`
}

type GetTagsResponse struct {
	Tags       []*entity.Tag      `json:"tags,omitempty"`
	Pagination *entity.Pagination `json:"pagination,omitempty"`
}

var GetTagsValidator = validator.MustForm(map[string]validator.Validator{
	"name":       ResourceNameValidator(true),
	"desc":       ResourceDescValidator(true),
	"pagination": PaginationValidator(),
})

func (h *tagHandler) GetTags(ctx context.Context, req *GetTagsRequest, res *GetTagsResponse) error {
	if err := GetTagsValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	tags, pagination, err := h.tagRepo.GetMany(ctx, &repo.TagFilter{
		Name: req.Name,
		Desc: req.Desc,
		Pagination: &repo.Pagination{
			Page:  goutil.Uint32(req.Pagination.GetPage()),
			Limit: goutil.Uint32(req.Pagination.GetLimit()),
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get tags failed: %v", err)
		return err
	}

	res.Tags = tags
	res.Pagination = pagination

	return nil
}

type CreateTagRequest struct {
	Name      *string           `json:"name,omitempty"`
	Desc      *string           `json:"desc,omitempty"`
	Enum      []string          `json:"enum,omitempty"`
	ValueType *uint32           `json:"value_type,omitempty"`
	ExtInfo   *CreateTagExtInfo `json:"ext_info,omitempty"`
}

type CreateTagExtInfo struct{}

func (req *CreateTagRequest) GetEnum() []string {
	if req != nil && req.Enum != nil {
		return req.Enum
	}
	return nil
}

func (req *CreateTagRequest) GetValueType() uint32 {
	if req != nil && req.ValueType != nil {
		return *req.ValueType
	}
	return 0
}

func (req *CreateTagRequest) ToTag() *entity.Tag {
	if req.ExtInfo == nil {
		req.ExtInfo = new(CreateTagExtInfo)
	}
	now := time.Now()
	return &entity.Tag{
		Name:       req.Name,
		Desc:       req.Desc,
		Enum:       req.Enum,
		ValueType:  req.ValueType,
		Status:     goutil.Uint32(uint32(entity.TagStatusNormal)),
		ExtInfo:    &entity.TagExtInfo{},
		CreateTime: goutil.Uint64(uint64(now.Unix())),
		UpdateTime: goutil.Uint64(uint64(now.Unix())),
	}
}

type CreateTagResponse struct {
	Tag *entity.Tag `json:"tag,omitempty"`
}

var CreateTagValidator = validator.MustForm(map[string]validator.Validator{
	"name": ResourceNameValidator(false),
	"desc": ResourceDescValidator(false),
	"value_type": &validator.UInt32{
		Optional:   false,
		Validators: []validator.UInt32Func{entity.CheckTagValueType},
	},
	"enum": &validator.Slice{
		Optional: true,
		MaxLen:   20,
	},
})

func (h *tagHandler) CreateTag(ctx context.Context, req *CreateTagRequest, res *CreateTagResponse) error {
	if err := CreateTagValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	tag := req.ToTag()
	for _, v := range tag.Enum {
		if ok := tag.IsValidTagValue(v); !ok {
			return errutil.ValidationError(errors.New("invalid tag value enum"))
		}
	}

	f := &repo.TagFilter{
		Name:   tag.Name,
		Status: goutil.Uint32(uint32(entity.TagStatusNormal)),
	}
	_, err := h.tagRepo.Get(ctx, f)
	if err == nil {
		return errors.New("tag already exists")
	}

	if !errors.Is(err, repo.ErrTagNotFound) {
		log.Ctx(ctx).Error().Msgf("get tag failed: %v", err)
		return err
	}

	id, err := h.tagRepo.Create(ctx, tag)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("create tag failed: %v", err)
		return err
	}

	tag.ID = goutil.Uint64(id)
	res.Tag = tag

	return nil
}
