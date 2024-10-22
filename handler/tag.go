package handler

import (
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"cdp/pkg/validator"
	"cdp/repo"
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"strconv"
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

type CreateTagExtInfo struct {
	DecimalPlace *uint32 `json:"decimal_place,omitempty"`
}

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
		Name:      req.Name,
		Desc:      req.Desc,
		Enum:      req.Enum,
		ValueType: req.ValueType,
		Status:    goutil.Uint32(uint32(entity.TagStatusNormal)),
		ExtInfo: &entity.TagExtInfo{
			DecimalPlace: req.ExtInfo.DecimalPlace,
		},
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
	c := NewTagCreator(tag.GetValueType())

	tag, err = c.PreCreate(ctx, tag)
	if err != nil {
		return errutil.ValidationError(err)
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

type TagCreator interface {
	PreCreate(ctx context.Context, tag *entity.Tag) (*entity.Tag, error)
}

func NewTagCreator(tagValueType uint32) TagCreator {
	switch tagValueType {
	case uint32(entity.TagValueTypeInt):
		return new(intTagCreator)
	case uint32(entity.TagValueTypeStr):
		return new(strTagCreator)
	case uint32(entity.TagValueTypeFloat):
		return new(floatTagCreator)
	}
	panic(fmt.Sprintf("tag creator not implemented for tag value type %v", tagValueType))
}

type strTagCreator struct{}

func (c *strTagCreator) PreCreate(_ context.Context, tag *entity.Tag) (*entity.Tag, error) {
	if tag.ExtInfo != nil && tag.ExtInfo.DecimalPlace != nil {
		return nil, entity.ErrDecimalPlaceNotAllowed
	}

	return tag, nil
}

type intTagCreator struct{}

func (c *intTagCreator) PreCreate(_ context.Context, tag *entity.Tag) (*entity.Tag, error) {
	for _, v := range tag.Enum {
		if _, err := strconv.Atoi(v); err != nil {
			return nil, fmt.Errorf("expect enum to be int, got %v", v)
		}
	}

	if tag.ExtInfo != nil && tag.ExtInfo.DecimalPlace != nil {
		return nil, entity.ErrDecimalPlaceNotAllowed
	}

	return tag, nil
}

type floatTagCreator struct{}

func (c *floatTagCreator) PreCreate(_ context.Context, tag *entity.Tag) (*entity.Tag, error) {
	if tag.ExtInfo == nil || tag.ExtInfo.GetDecimalPlace() == 0 {
		return nil, errors.New("expect decimal place to be non-empty")
	}

	if tag.ExtInfo.GetDecimalPlace() > entity.MaxDecimalPlace {
		return nil, errors.New("exceed max decimal place")
	}

	// format enum to the right decimal place
	for i, e := range tag.Enum {
		s, err := goutil.FormatFloat(e, tag.ExtInfo.GetDecimalPlace())
		if err != nil {
			return nil, fmt.Errorf("expect enum to be float, got %v", e)
		}
		tag.Enum[i] = s
	}

	return tag, nil
}
