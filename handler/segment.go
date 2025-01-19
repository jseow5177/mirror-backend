package handler

import (
	"cdp/config"
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
	GetUds(ctx context.Context, req *GetUdsRequest, res *GetUdsResponse) error
	GetSegment(ctx context.Context, req *GetSegmentRequest, res *GetSegmentResponse) error
	GetSegments(ctx context.Context, req *GetSegmentsRequest, res *GetSegmentsResponse) error
	PreviewUd(ctx context.Context, req *PreviewUdRequest, res *PreviewUdResponse) error
	CountSegments(ctx context.Context, req *CountSegmentsRequest, res *CountSegmentsResponse) error
}

type segmentHandler struct {
	cfg         *config.Config
	segmentRepo repo.SegmentRepo
	tagRepo     repo.TagRepo
}

func NewSegmentHandler(cfg *config.Config, tagRepo repo.TagRepo, segmentRepo repo.SegmentRepo) SegmentHandler {
	return &segmentHandler{
		cfg:         cfg,
		tagRepo:     tagRepo,
		segmentRepo: segmentRepo,
	}
}

type CountSegmentsRequest struct {
	ContextInfo
}

type CountSegmentsResponse struct {
	Count *uint64 `json:"count,omitempty"`
}

var CountSegmentsValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator,
})

func (h *segmentHandler) CountSegments(ctx context.Context, req *CountSegmentsRequest, res *CountSegmentsResponse) error {
	if err := CountSegmentsValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	count, err := h.segmentRepo.CountByTenantID(ctx, req.GetTenantID())
	if err != nil {
		return err
	}

	res.Count = goutil.Uint64(count)

	return nil
}

type GetSegmentRequest struct {
	ContextInfo

	SegmentID *uint64 `json:"segment_id,omitempty"`
}

func (r *GetSegmentRequest) GetSegmentID() uint64 {
	if r != nil && r.SegmentID != nil {
		return *r.SegmentID
	}
	return 0
}

type GetSegmentResponse struct {
	Segment *entity.Segment `json:"segment,omitempty"`
}

var GetSegmentValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator,
	"segment_id":  &validator.UInt64{},
})

func (h *segmentHandler) GetSegment(ctx context.Context, req *GetSegmentRequest, res *GetSegmentResponse) error {
	if err := GetSegmentValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	segment, err := h.segmentRepo.GetByID(ctx, req.GetTenantID(), req.GetSegmentID())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get segment err: %v", err)
		return err
	}

	res.Segment = segment

	return nil
}

type GetSegmentsRequest struct {
	ContextInfo

	Keyword    *string          `json:"keyword,omitempty"`
	Pagination *repo.Pagination `json:"pagination,omitempty"`
}

func (r *GetSegmentsRequest) GetKeyword() string {
	if r != nil && r.Keyword != nil {
		return *r.Keyword
	}
	return ""
}

type GetSegmentsResponse struct {
	Segments   []*entity.Segment `json:"segments"`
	Pagination *repo.Pagination  `json:"pagination,omitempty"`
}

var GetSegmentsValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator,
	"keyword": &validator.String{
		Optional: true,
	},
	"pagination": PaginationValidator(),
})

func (h *segmentHandler) GetSegments(ctx context.Context, req *GetSegmentsRequest, res *GetSegmentsResponse) error {
	if err := GetSegmentsValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	if req.Pagination == nil {
		req.Pagination = new(repo.Pagination)
	}

	segments, pagination, err := h.segmentRepo.GetByKeyword(ctx, req.GetTenantID(), req.GetKeyword(), req.Pagination)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get segments failed: %v", err)
		return err
	}

	res.Segments = segments
	res.Pagination = pagination

	return nil
}

type CreateSegmentRequest struct {
	ContextInfo

	Name        *string       `json:"name,omitempty"`
	SegmentDesc *string       `json:"segment_desc,omitempty"`
	Criteria    *entity.Query `json:"criteria,omitempty"`
}

func (req *CreateSegmentRequest) ToSegment() *entity.Segment {
	if req.Criteria == nil {
		req.Criteria = new(entity.Query)
	}
	now := time.Now()
	return &entity.Segment{
		Name:        req.Name,
		SegmentDesc: req.SegmentDesc,
		Criteria:    req.Criteria,
		Status:      entity.SegmentStatusNormal,
		CreatorID:   req.User.ID,
		TenantID:    req.Tenant.ID,
		CreateTime:  goutil.Uint64(uint64(now.Unix())),
		UpdateTime:  goutil.Uint64(uint64(now.Unix())),
	}
}

type CreateSegmentResponse struct {
	Segment *entity.Segment `json:"segment,omitempty"`
}

var CreateSegmentValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo":  ContextInfoValidator,
	"name":         ResourceNameValidator(false),
	"segment_desc": ResourceDescValidator(false),
})

func (h *segmentHandler) CreateSegment(ctx context.Context, req *CreateSegmentRequest, res *CreateSegmentResponse) error {
	if err := CreateSegmentValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	v := NewQueryValidator(req.GetTenantID(), h.tagRepo, false)
	if err := v.Validate(ctx, req.Criteria); err != nil {
		return errutil.ValidationError(err)
	}

	segment := req.ToSegment()

	_, err := h.segmentRepo.GetByName(ctx, req.GetTenantID(), segment.GetName())
	if err == nil {
		return errutil.ConflictError(errors.New("segment already exists"))
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

type GetUdsRequest struct {
	ContextInfo

	SegmentID *uint64 `json:"segment_id,omitempty"`
}

func (req *GetUdsRequest) GetSegmentID() uint64 {
	if req != nil && req.SegmentID != nil {
		return *req.SegmentID
	}
	return 0
}

type GetUdsResponse struct {
	Uds []*entity.Ud `json:"uds"`
}

var GetUdsValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator,
	"segment_id": &validator.UInt64{
		Optional: false,
	},
})

func (h *segmentHandler) GetUds(ctx context.Context, req *GetUdsRequest, res *GetUdsResponse) error {
	if err := GetUdsValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	_, err := h.segmentRepo.GetByID(ctx, req.GetTenantID(), req.GetSegmentID())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get segment failed: %v", err)
		return err
	}

	uds := make([]*entity.Ud, 0)
	for _, email := range h.cfg.TestEmails {
		uds = append(uds, &entity.Ud{
			ID:     goutil.String(email),
			IDType: entity.IDTypeEmail,
		})
	}

	res.Uds = uds

	return nil
}

type CountUdRequest struct {
	ContextInfo

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
	"ContextInfo": ContextInfoValidator,
	"segment_id": &validator.UInt64{
		Optional: false,
	},
})

func (h *segmentHandler) CountUd(ctx context.Context, req *CountUdRequest, res *CountUdResponse) error {
	if err := CountUdValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	_, err := h.segmentRepo.GetByID(ctx, req.GetTenantID(), req.GetSegmentID())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get segment failed: %v", err)
		return err
	}

	res.Count = goutil.Uint64(0)

	return nil
}

type PreviewUdRequest struct {
	ContextInfo

	Criteria *entity.Query `json:"criteria,omitempty"`
}

type PreviewUdResponse struct {
	Count *int64 `json:"count,omitempty"`
}

var PreviewUdValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator,
})

func (h *segmentHandler) PreviewUd(ctx context.Context, req *PreviewUdRequest, res *PreviewUdResponse) error {
	if err := PreviewUdValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	v := NewQueryValidator(req.GetTenantID(), h.tagRepo, false)
	if err := v.Validate(ctx, req.Criteria); err != nil {
		log.Ctx(ctx).Warn().Msgf("validate segment failed: %v", err)
		res.Count = goutil.Int64(-1)
		return nil
	}

	res.Count = goutil.Int64(0)

	return nil
}
