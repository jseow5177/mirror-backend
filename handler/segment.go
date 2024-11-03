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

type SegmentHandler interface {
	CreateSegment(ctx context.Context, req *CreateSegmentRequest, res *CreateSegmentResponse) error
	CountUd(ctx context.Context, req *CountUdRequest, res *CountUdResponse) error
}

type segmentHandler struct {
	segmentRepo repo.SegmentRepo
	tagRepo     repo.TagRepo
	queryRepo   repo.QueryRepo
}

func NewSegmentHandler(tagRepo repo.TagRepo, segmentRepo repo.SegmentRepo, queryRepo repo.QueryRepo) SegmentHandler {
	return &segmentHandler{
		tagRepo:     tagRepo,
		segmentRepo: segmentRepo,
		queryRepo:   queryRepo,
	}
}

type CreateSegmentRequest struct {
	Name  *string       `json:"name,omitempty"`
	Desc  *string       `json:"desc,omitempty"`
	Query *entity.Query `json:"query,omitempty"`
}

func (req *CreateSegmentRequest) ToSegment() *entity.Segment {
	if req.Query == nil {
		req.Query = new(entity.Query)
	}
	now := time.Now()
	return &entity.Segment{
		Name:       req.Name,
		Desc:       req.Desc,
		Query:      req.Query,
		Status:     goutil.Uint32(uint32(entity.SegmentStatusNormal)),
		CreateTime: goutil.Uint64(uint64(now.Unix())),
		UpdateTime: goutil.Uint64(uint64(now.Unix())),
	}
}

type CreateSegmentResponse struct {
	Segment *entity.Segment `json:"segment,omitempty"`
}

var CreateSegmentValidator = validator.MustForm(map[string]validator.Validator{
	"name": ResourceNameValidator(false),
	"desc": ResourceDescValidator(false),
})

func (h *segmentHandler) CreateSegment(ctx context.Context, req *CreateSegmentRequest, res *CreateSegmentResponse) error {
	if err := CreateSegmentValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	v := NewQueryValidator(h.tagRepo, false)
	if err := v.Validate(ctx, req.Query); err != nil {
		return errutil.ValidationError(err)
	}

	segment := req.ToSegment()

	f := &repo.SegmentFilter{
		Name:   segment.Name,
		Status: goutil.Uint32(uint32(entity.SegmentStatusNormal)),
	}
	_, err := h.segmentRepo.Get(ctx, f)
	if err == nil {
		return errors.New("segment already exists")
	}

	if !errors.Is(err, repo.ErrSegmentNotFound) {
		log.Ctx(ctx).Error().Msgf("get segment failed: %v", err)
		return err
	}

	id, err := h.segmentRepo.Create(ctx, segment)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("create segment failed: %v", err)
		return err
	}

	segment.ID = goutil.Uint64(id)
	res.Segment = segment

	return nil
}

type CountUdRequest struct {
	SegmentID *uint64 `json:"segment_id,omitempty"`
}

func (req *CountUdRequest) GetSegmentID() uint64 {
	if req != nil && req.SegmentID != nil {
		return *req.SegmentID
	}
	return 0
}

type CountUdResponse struct {
	Count *uint64 `json:"count,omitempty"`
}

var CountUdValidator = validator.MustForm(map[string]validator.Validator{
	"segment_id": &validator.UInt64{
		Optional: false,
	},
})

func (h *segmentHandler) CountUd(ctx context.Context, req *CountUdRequest, res *CountUdResponse) error {
	if err := CountUdValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	segment, err := h.segmentRepo.Get(ctx, &repo.SegmentFilter{
		ID: req.SegmentID,
	})
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get segment failed: %v", err)
		return err
	}

	count, err := h.queryRepo.Count(ctx, segment.Query)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get count failed: %v", err)
		return err
	}

	res.Count = goutil.Uint64(count)

	return nil
}
