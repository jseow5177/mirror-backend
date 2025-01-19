package handler

import (
	"cdp/entity"
	"cdp/pkg/goutil"
	"cdp/pkg/router"
	"cdp/pkg/validator"
	"cdp/repo"
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
)

const MaxQueryDepth = 5

func PaginationValidator() validator.Validator {
	return validator.OptionalForm(map[string]validator.Validator{
		"page": &validator.UInt32{
			Optional: true,
		},
		"limit": &validator.UInt32{
			Optional: true,
		},
	})
}

func CheckIDType(idType uint32) error {
	if _, ok := entity.IDTypes[entity.IDType(idType)]; ok && idType != uint32(entity.IDTypeUnknown) {
		return nil
	}
	return errors.New("invalid id type")
}

func UDValidator() validator.Validator {
	return validator.MustForm(map[string]validator.Validator{
		"id": &validator.String{},
		"id_type": &validator.UInt32{
			Optional:   true,
			Validators: []validator.UInt32Func{CheckIDType},
		},
	})
}

func UdTagValValidator() validator.Validator {
	return validator.MustForm(map[string]validator.Validator{
		"ud": UDValidator(),
		"tag_vals": &validator.Slice{
			MinLen: 1,
			MaxLen: 50,
			Validator: validator.MustForm(map[string]validator.Validator{
				"tag_id":   &validator.UInt64{Optional: true},
				"tag_name": ResourceNameValidator(true),
				"tag_val":  &validator.String{Optional: true, MaxLen: 8192},
			}),
		},
	})
}

func PasswordValidator(optional bool) validator.Validator {
	return &validator.String{
		Optional:  optional,
		UnsetZero: true,
		MinLen:    8,
		MaxLen:    128,
	}
}

func DisplayNameValidator(optional bool) validator.Validator {
	return &validator.String{
		Optional:  optional,
		UnsetZero: true,
		MaxLen:    100,
		Regex:     regexp.MustCompile(`^[a-zA-Z0-9_.\s-]+$`),
	}
}

func EmailValidator(optional bool) validator.Validator {
	return &validator.String{
		Optional:  optional,
		UnsetZero: true,
		MaxLen:    254,
		Validators: []validator.StringFunc{
			func(s string) error {
				_, err := mail.ParseAddress(s)
				return err
			},
		},
	}
}

func ResourceNameValidator(optional bool) validator.Validator {
	return &validator.String{
		Optional:  optional,
		UnsetZero: true,
		MaxLen:    60,
		Regex:     regexp.MustCompile(`^[0-9a-zA-Z_.\s]+$`),
	}
}

func ResourceDescValidator(optional bool) validator.Validator {
	return &validator.String{
		Optional: optional,
		MaxLen:   200,
	}
}

type QueryValidator interface {
	Validate(ctx context.Context, query *entity.Query) error
}

type queryValidator struct {
	tenantID uint64
	tagRepo  repo.TagRepo
	optional bool
}

func NewQueryValidator(tenantID uint64, tagRepo repo.TagRepo, optional bool) QueryValidator {
	return &queryValidator{
		tenantID: tenantID,
		tagRepo:  tagRepo,
		optional: optional,
	}
}

func (v *queryValidator) Validate(ctx context.Context, query *entity.Query) error {
	if query == nil {
		if !v.optional {
			return errors.New("missing query")
		}
	} else {
		if err := v.validateQuery(ctx, query, 0); err != nil {
			return err
		}
	}

	return nil
}

func (v *queryValidator) validateQuery(ctx context.Context, query *entity.Query, depth int) error {
	if query == nil {
		return nil
	}

	if depth > MaxQueryDepth {
		return fmt.Errorf("query depth exceeds max depth (%d)", MaxQueryDepth)
	}

	if !goutil.MustHaveOne(query.Queries, query.Lookups) {
		return errors.New("query cannot have both queries and lookups")
	}

	if query.GetOp() != entity.QueryOpAnd && query.GetOp() != entity.QueryOpOr {
		return fmt.Errorf("invalid query op, only %s or %s are supported", entity.QueryOpAnd, entity.QueryOpOr)
	}

	for _, query := range query.Queries {
		if err := v.validateQuery(ctx, query, depth+1); err != nil {
			return err
		}
	}

	for _, lookup := range query.Lookups {
		if err := v.validateLookup(ctx, lookup); err != nil {
			return err
		}
	}

	return nil
}

func (v *queryValidator) validateLookup(ctx context.Context, lookup *entity.Lookup) error {
	if lookup == nil {
		return nil
	}

	if lookup.TagID == nil {
		return errors.New("missing tag id in lookup")
	}

	tag, err := v.tagRepo.GetByID(ctx, v.tenantID, lookup.GetTagID())
	if err != nil {
		return err
	}

	if !goutil.MustHaveOne(lookup.In, lookup.Eq, lookup.Range) {
		return errors.New("lookup can have only one of in, eq, or range")
	}

	const errTmpl = "lookup tag value %s is invalid"

	if lookup.Eq != nil {
		if ok := tag.IsValidTagValue(lookup.GetEq()); !ok {
			return fmt.Errorf(errTmpl, lookup.GetEq())
		}
	}

	if lookup.In != nil {
		if len(lookup.In) == 0 {
			return fmt.Errorf("include cannot be empty")
		}
		for _, in := range lookup.In {
			if ok := tag.IsValidTagValue(in); !ok {
				return fmt.Errorf(errTmpl, in)
			}
		}
	}

	if lookup.Range != nil {
		if !tag.IsNumeric() {
			return errors.New("only numeric tag can have range lookup")
		}

		if !goutil.AtLeastOne(lookup.Range.Gt, lookup.Range.Gte, lookup.Range.Lt, lookup.Range.Lte) {
			return errors.New("lookup range must have one of Gt, Gte, Lt, and Lte")
		}

		if !goutil.AtMostOne(lookup.Range.Gt, lookup.Range.Gte) {
			return errors.New("lookup range can only one of Gt or Gte")
		}

		if !goutil.AtMostOne(lookup.Range.Lt, lookup.Range.Lte) {
			return errors.New("lookup range can only one of Lt or Lte")
		}

		if lookup.Range.Gt != nil {
			if ok := tag.IsValidTagValue(lookup.Range.GetGt()); !ok {
				return fmt.Errorf(errTmpl, lookup.Range.GetGt())
			}
		}

		if lookup.Range.Gte != nil {
			if ok := tag.IsValidTagValue(lookup.Range.GetGte()); !ok {
				return fmt.Errorf(errTmpl, lookup.Range.GetGte())
			}
		}

		if lookup.Range.Lt != nil {
			if ok := tag.IsValidTagValue(lookup.Range.GetLt()); !ok {
				return fmt.Errorf(errTmpl, lookup.Range.GetLt())
			}
		}

		if lookup.Range.Lte != nil {
			if ok := tag.IsValidTagValue(lookup.Range.GetLte()); !ok {
				return fmt.Errorf(errTmpl, lookup.Range.GetLte())
			}
		}
	}

	return nil
}

type fileInfoValidator struct {
	maxSize     int64
	contentType []string
	optional    bool
}

func (v *fileInfoValidator) Validate(value interface{}) error {
	fileInfo, ok := value.(*router.FileMeta)
	if !ok {
		return errors.New("expect FileInfo")
	}

	if fileInfo == nil || fileInfo.File == nil {
		if !v.optional {
			return errors.New("missing file")
		}
	} else {
		if fileInfo.FileHeader.Size > v.maxSize {
			return errors.New("file size too large")
		}
		if len(v.contentType) > 0 && !goutil.ContainsStr(v.contentType, fileInfo.FileHeader.Header.Get("Content-Type")) {
			return errors.New("invalid file type")
		}
	}

	return nil
}

func FileInfoValidator(optional bool, maxSize int64, contentType []string) validator.Validator {
	return &fileInfoValidator{
		optional:    optional,
		maxSize:     maxSize,
		contentType: contentType,
	}
}
