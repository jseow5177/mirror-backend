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
	GetSegments(ctx context.Context, req *GetSegmentsRequest, res *GetSegmentsResponse) error
	PreviewUd(ctx context.Context, req *PreviewUdRequest, res *PreviewUdResponse) error
	CountSegments(ctx context.Context, req *CountSegmentsRequest, res *CountSegmentsResponse) error
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

type CountSegmentsRequest struct{}

type CountSegmentsResponse struct {
	Count *uint64 `json:"count,omitempty"`
}

var CountSegmentsValidator = validator.MustForm(map[string]validator.Validator{})

func (h *segmentHandler) CountSegments(ctx context.Context, req *CountSegmentsRequest, res *CountSegmentsResponse) error {
	if err := CountSegmentsValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	count, err := h.segmentRepo.Count(ctx, nil)
	if err != nil {
		return err
	}

	res.Count = goutil.Uint64(count)

	return nil
}

type GetSegmentsRequest struct {
	Name       *string            `json:"name,omitempty"`
	Desc       *string            `json:"desc,omitempty"`
	Pagination *entity.Pagination `json:"pagination,omitempty"`
}

type GetSegmentsResponse struct {
	Segments   []*entity.Segment  `json:"segments,omitempty"`
	Pagination *entity.Pagination `json:"pagination,omitempty"`
}

var GetSegmentsValidator = validator.MustForm(map[string]validator.Validator{
	"name":       ResourceNameValidator(true),
	"desc":       ResourceDescValidator(true),
	"pagination": PaginationValidator(),
})

func (h *segmentHandler) GetSegments(ctx context.Context, req *GetSegmentsRequest, res *GetSegmentsResponse) error {
	if err := GetSegmentsValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	segments, pagination, err := h.segmentRepo.GetMany(ctx, &repo.SegmentFilter{
		Name: req.Name,
		Desc: req.Desc,
		Pagination: &repo.Pagination{
			Page:  req.Pagination.Page,
			Limit: req.Pagination.Limit,
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get segments failed: %v", err)
		return err
	}

	res.Segments = segments
	res.Pagination = pagination

	return nil
}

type CreateSegmentRequest struct {
	Name     *string       `json:"name,omitempty"`
	Desc     *string       `json:"desc,omitempty"`
	Criteria *entity.Query `json:"criteria,omitempty"`
}

func (req *CreateSegmentRequest) ToSegment() *entity.Segment {
	if req.Criteria == nil {
		req.Criteria = new(entity.Query)
	}
	now := time.Now()
	return &entity.Segment{
		Name:       req.Name,
		Desc:       req.Desc,
		Criteria:   req.Criteria,
		Status:     entity.SegmentStatusNormal,
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
	if err := v.Validate(ctx, req.Criteria); err != nil {
		return errutil.ValidationError(err)
	}

	segment := req.ToSegment()

	f := &repo.SegmentFilter{
		Name:   segment.Name,
		Status: goutil.Uint32(uint32(entity.SegmentStatusNormal)),
	}
	_, err := h.segmentRepo.Get(ctx, f)
	if err == nil {
		return errutil.ValidationError(errors.New("segment already exists"))
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

	count, err := h.queryRepo.Count(ctx, segment.Criteria)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get count failed: %v", err)
		return err
	}

	res.Count = goutil.Uint64(count)

	return nil
}

type PreviewUdRequest struct {
	Criteria *entity.Query `json:"criteria,omitempty"`
}

type PreviewUdResponse struct {
	Count *int64 `json:"count,omitempty"`
}

var PreviewUdValidator = validator.MustForm(map[string]validator.Validator{})

func (h *segmentHandler) PreviewUd(ctx context.Context, req *PreviewUdRequest, res *PreviewUdResponse) error {
	if err := PreviewUdValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	v := NewQueryValidator(h.tagRepo, false)
	if err := v.Validate(ctx, req.Criteria); err != nil {
		log.Ctx(ctx).Warn().Msgf("validate segment failed: %v", err)
		res.Count = goutil.Int64(-1)
		return nil
	}

	count, err := h.queryRepo.Count(ctx, req.Criteria)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get preview count failed: %v", err)
		return err
	}

	res.Count = goutil.Int64(int64(count))

	return nil
}
