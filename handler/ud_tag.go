package handler

import (
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/validator"
	"context"
)

type UdTagHandler interface {
	SetUdTag(ctx context.Context, req *SetUdTagRequest, resp *SetUdTagResponse) error
}

type udTagHandler struct{}

type SetUdTagRequest struct {
	UdTagVal *entity.UdTagVal `json:"ud_tag_val,omitempty"`
	Token    *string          `json:"token,omitempty"`
}

type SetUdTagResponse struct{}

var SetUdTagValidator = validator.MustForm(map[string]validator.Validator{
	"ud_tag_val": UdTagValValidator(),
	"token": &validator.String{
		Optional: true,
	},
})

func (h *udTagHandler) SetUdTag(ctx context.Context, req *SetUdTagRequest, resp *SetUdTagResponse) error {
	if err := SetUdTagValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	return nil
}
