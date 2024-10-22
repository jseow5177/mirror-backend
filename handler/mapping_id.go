package handler

import (
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"cdp/pkg/validator"
	"cdp/repo"
	"context"
	"github.com/rs/zerolog/log"
)

const (
	MappingIDsMaxSize = 50
)

type MappingIDHandler interface {
	GetMappingIDs(ctx context.Context, req *GetMappingIDsRequest, res *GetMappingIDsResponse) error
	GetSetMappingIDs(ctx context.Context, req *GetSetMappingIDsRequest, res *GetSetMappingIDsResponse) error
}

type mappingIDHandler struct {
	mappingIDRepo repo.MappingIDRepo
}

func NewMappingIDHandler(mappingIDRepo repo.MappingIDRepo) MappingIDHandler {
	return &mappingIDHandler{
		mappingIDRepo: mappingIDRepo,
	}
}

type GetMappingIDsRequest struct {
	UdIDs []string `json:"ud_ids,omitempty"`
}

type GetMappingIDsResponse struct {
	MappingIDs []*entity.MappingID `json:"mapping_ids"`
}

var GetMappingIDsValidator = validator.MustForm(map[string]validator.Validator{
	"ud_ids": &validator.Slice{
		Optional: false,
		MinLen:   1,
		MaxLen:   MappingIDsMaxSize,
	},
})

func (h *mappingIDHandler) GetMappingIDs(ctx context.Context, req *GetMappingIDsRequest, res *GetMappingIDsResponse) error {
	if err := GetMappingIDsValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	mappingIDs, err := h.mappingIDRepo.GetMany(ctx, req.UdIDs)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to get mapping IDs: %s", err.Error())
		return err
	}

	mappingIDsMap := make(map[string]*entity.MappingID, len(mappingIDs))
	for _, mappingID := range mappingIDs {
		mappingIDsMap[mappingID.GetUdID()] = mappingID
	}

	resMappingIDs := make([]*entity.MappingID, 0, len(req.UdIDs))
	for _, udID := range req.UdIDs {
		var id uint64

		mappingID := mappingIDsMap[udID]
		if mappingID != nil {
			id = mappingID.GetID()
		}

		resMappingIDs = append(resMappingIDs, &entity.MappingID{
			ID:   goutil.Uint64(id),
			UdID: goutil.String(udID),
		})
	}

	res.MappingIDs = resMappingIDs

	return nil
}

type GetSetMappingIDsRequest struct {
	UdIDs []string `json:"ud_ids,omitempty"`
}

type GetSetMappingIDsResponse struct {
	MappingIDs []*entity.MappingID `json:"mapping_ids"`
}

var GetSetMappingIDsValidator = validator.MustForm(map[string]validator.Validator{
	"ud_ids": &validator.Slice{
		Optional: false,
		MinLen:   1,
		MaxLen:   MappingIDsMaxSize,
	},
})

func (h *mappingIDHandler) GetSetMappingIDs(ctx context.Context, req *GetSetMappingIDsRequest, res *GetSetMappingIDsResponse) error {
	if err := GetSetMappingIDsValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	getReq := &GetMappingIDsRequest{
		UdIDs: req.UdIDs,
	}
	getRes := new(GetMappingIDsResponse)

	if err := h.GetMappingIDs(ctx, getReq, getRes); err != nil {
		log.Ctx(ctx).Error().Msgf("failed to get mapping IDs: %s", err.Error())
		return err
	}

	resMappingIDs := getRes.MappingIDs

	var (
		notFoundMappingIDs = make([]*entity.MappingID, 0, len(resMappingIDs))
		notFoundUdIDs      = make([]string, 0, len(resMappingIDs))
	)
	for _, mappingID := range resMappingIDs {
		if mappingID.GetID() == 0 {
			notFoundMappingIDs = append(notFoundMappingIDs, mappingID)
			notFoundUdIDs = append(notFoundUdIDs, mappingID.GetUdID())
		}
	}

	if len(notFoundMappingIDs) == 0 {
		res.MappingIDs = resMappingIDs
		return nil
	}

	if err := h.mappingIDRepo.CreateMany(ctx, notFoundMappingIDs); err != nil {
		log.Ctx(ctx).Error().Msgf("failed to create mapping IDs: %s", err.Error())
		return err
	}

	// get newly created mapping IDs
	getReq = &GetMappingIDsRequest{
		UdIDs: notFoundUdIDs,
	}
	getRes = new(GetMappingIDsResponse)
	if err := h.GetMappingIDs(ctx, getReq, getRes); err != nil {
		log.Ctx(ctx).Error().Msgf("failed to get newly created mapping IDs: %s", err.Error())
		return err
	}

	// set result
	var idx int
	for _, mappingID := range resMappingIDs {
		if mappingID.GetID() == 0 {
			mappingID.ID = getRes.MappingIDs[idx].ID
			idx++
		}
	}

	res.MappingIDs = resMappingIDs

	return nil
}
