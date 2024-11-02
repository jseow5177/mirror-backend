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
}

type tagHandler struct {
	tagRepo repo.TagRepo
}

func NewTagHandler(tagRepo repo.TagRepo) TagHandler {
	return &tagHandler{
		tagRepo: tagRepo,
	}
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
	"name": TagNameValidator(false),
	"desc": &validator.String{
		Optional: false,
		MaxLen:   200,
	},
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
	err := CreateTagValidator.Validate(req)
	if err != nil {
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
	if _, err := h.tagRepo.Get(ctx, f); err != nil {
		if errors.Is(err, repo.ErrTagNotFound) {
			id, err := h.tagRepo.Create(ctx, tag)
			if err != nil {
				log.Ctx(ctx).Error().Msgf("create tag failed: %v", err)
				return err
			}

			tag.ID = goutil.Uint64(id)
			res.Tag = tag

			return nil
		}
		log.Ctx(ctx).Error().Msgf("get tag failed: %v", err)
		return err
	}

	return errors.New("tag already exists")
}
