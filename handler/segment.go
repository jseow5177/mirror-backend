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
	DownloadUds(ctx context.Context, req *DownloadUdsRequest, res *DownloadUdsResponse) error
	GetSegment(ctx context.Context, req *GetSegmentRequest, res *GetSegmentResponse) error
	GetSegments(ctx context.Context, req *GetSegmentsRequest, res *GetSegmentsResponse) error
	PreviewUd(ctx context.Context, req *PreviewUdRequest, res *PreviewUdResponse) error
	CountSegments(ctx context.Context, req *CountSegmentsRequest, res *CountSegmentsResponse) error
}

type segmentHandler struct {
	cfg         *config.Config
	segmentRepo repo.SegmentRepo
	tagRepo     repo.TagRepo
	queryRepo   repo.QueryRepo
}

func NewSegmentHandler(cfg *config.Config, tagRepo repo.TagRepo, segmentRepo repo.SegmentRepo, queryRepo repo.QueryRepo) SegmentHandler {
	return &segmentHandler{
		cfg:         cfg,
		tagRepo:     tagRepo,
		segmentRepo: segmentRepo,
		queryRepo:   queryRepo,
	}
}

type CountSegmentsRequest struct {
	ContextInfo
}

type CountSegmentsResponse struct {
	Count *uint64 `json:"count,omitempty"`
}

var CountSegmentsValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator(false, true),
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
	"ContextInfo": ContextInfoValidator(false, true),
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
	"ContextInfo": ContextInfoValidator(false, true),
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
		CreatorID:   goutil.Uint64(req.GetUserID()),
		TenantID:    goutil.Uint64(req.GetTenantID()),
		CreateTime:  goutil.Uint64(uint64(now.Unix())),
		UpdateTime:  goutil.Uint64(uint64(now.Unix())),
	}
}

type CreateSegmentResponse struct {
	Segment *entity.Segment `json:"segment,omitempty"`
}

var CreateSegmentValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo":  ContextInfoValidator(false, false),
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

type DownloadUdsRequest struct {
	ContextInfo

	SegmentID  *uint64          `json:"segment_id,omitempty"`
	Pagination *repo.Pagination `json:"pagination,omitempty"`
}

func (req *DownloadUdsRequest) GetSegmentID() uint64 {
	if req != nil && req.SegmentID != nil {
		return *req.SegmentID
	}
	return 0
}

type DownloadUdsResponse struct {
	Uds        []*entity.Ud     `json:"uds"`
	Pagination *repo.Pagination `json:"pagination,omitempty"`
}

func (res *DownloadUdsResponse) GetPagination() *repo.Pagination {
	if res != nil && res.Pagination != nil {
		return res.Pagination
	}
	return nil
}

var DownloadUdsValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator(false, true),
	"segment_id": &validator.UInt64{
		Optional: false,
	},
	"pagination": PaginationValidator(),
})

func (h *segmentHandler) DownloadUds(ctx context.Context, req *DownloadUdsRequest, res *DownloadUdsResponse) error {
	if err := DownloadUdsValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	segment, err := h.segmentRepo.GetByID(ctx, req.GetTenantID(), req.GetSegmentID())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get segment failed: %v", err)
		return err
	}

	if req.Pagination == nil {
		req.Pagination = &repo.Pagination{
			Limit:  goutil.Uint32(DefaultMaxLimit),
			Cursor: goutil.String(""),
		}
	}

	uds, newPage, err := h.queryRepo.Download(ctx, req.GetTenantName(), segment.GetCriteria(), req.Pagination)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("download uds failed: %v", err)
		return err
	}

	res.Uds = uds
	res.Pagination = newPage

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
	"ContextInfo": ContextInfoValidator(false, true),
	"segment_id": &validator.UInt64{
		Optional: false,
	},
})

func (h *segmentHandler) CountUd(ctx context.Context, req *CountUdRequest, res *CountUdResponse) error {
	if err := CountUdValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	segment, err := h.segmentRepo.GetByID(ctx, req.GetTenantID(), req.GetSegmentID())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get segment failed: %v", err)
		return err
	}

	count, err := h.queryRepo.Count(ctx, req.GetTenantName(), segment.GetCriteria())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get segment count failed: %v", err)
		return err
	}

	res.Count = goutil.Uint64(count)

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
	"ContextInfo": ContextInfoValidator(false, true),
})

func (h *segmentHandler) PreviewUd(ctx context.Context, req *PreviewUdRequest, res *PreviewUdResponse) error {
	if err := PreviewUdValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	v := NewQueryValidator(req.GetTenantID(), h.tagRepo, false)
	if err := v.Validate(ctx, req.Criteria); err != nil {
		res.Count = goutil.Int64(-1)
		return nil
	}

	count, err := h.queryRepo.Count(ctx, req.GetTenantName(), req.Criteria)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("preview segment count failed: %v", err)
		return err
	}

	res.Count = goutil.Int64(int64(count))

	return nil
}
