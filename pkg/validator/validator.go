package validator

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

type Validator interface {
	Validate(value interface{}) error
}

// ========== Used to unset zero values ========== //

type defaultValue struct {
	value interface{}
}

func (uv *defaultValue) Error() string {
	return ""
}

func (uv *defaultValue) GetValue() interface{} {
	return uv.value
}

// ========== Form Field Validator ========== //

type Form struct {
	Optional   bool
	Validators map[string]Validator
}

func MustForm(validators map[string]Validator) Validator {
	return &Form{
		Validators: validators,
	}
}

func OptionalForm(validators map[string]Validator) Validator {
	return &Form{
		Optional:   true,
		Validators: validators,
	}
}

func (f *Form) Validate(form interface{}) error {
	if form == nil {
		if f.Optional {
			return nil
		}
		return errors.New("form is required")
	}

	// no check needed
	if f.Validators == nil || len(f.Validators) == 0 {
		return nil
	}

	// form must be a struct
	sv := reflect.Indirect(reflect.ValueOf(form))
	if sv.Kind() != reflect.Struct {
		return errors.New("form must be a struct")
	}

	// validate each struct field
	for i := 0; i < sv.NumField(); i++ {
		fv := sv.Field(i)
		fn := getFieldName(sv.Type().Field(i))

		if validator, ok := f.Validators[fn]; ok && validator != nil {
			var err error

			kind := fv.Kind()
			for {
				if kind == reflect.Ptr {
					if fv.IsNil() {
						err = validator.Validate(nil)
						break
					}

					// nested struct
					if fv.Elem().Kind() == reflect.Struct {
						err = validator.Validate(fv.Interface())
						break
					}

					// scalar types
					err = validator.Validate(reflect.Indirect(fv).Interface())
					var val *defaultValue
					if errors.As(err, &val) {
						if val.GetValue() == nil {
							fv.Set(reflect.Zero(fv.Type()))
						} else {
							reflect.Indirect(fv).Set(reflect.ValueOf(val.GetValue()))
						}
						err = nil
					}
					break
				}

				if kind == reflect.Slice {
					if fv.IsNil() {
						err = validator.Validate(nil)
						break
					}

					err = validator.Validate(fv.Interface())
					break
				}

				// unknown kind
				if fv.CanInterface() {
					err = validator.Validate(fv.Interface())
					break
				}

				err = fmt.Errorf("unsupported kind or value: %v", fv)
				break
			}
			if err != nil {
				return fmt.Errorf("%s: %v", fn, err)
			}
		}
	}

	return nil
}

func getFieldName(structField reflect.StructField) string {
	jsonTag := structField.Tag.Get("json")
	switch jsonTag {
	case "-":
		return ""
	case "":
		return structField.Name
	default:
		parts := strings.Split(jsonTag, ",")
		if len(parts) == 0 {
			return structField.Name
		}

		jsonFieldName := parts[0]
		if jsonFieldName == "" {
			return structField.Name
		}

		return jsonFieldName
	}
}

// ========== Slice Field Validator ========== //

type Slice struct {
	Optional  bool
	MinLen    uint32
	MaxLen    uint32
	Validator Validator
}

func (sv *Slice) Validate(value interface{}) error {
	if value == nil {
		if sv.Optional {
			return nil
		}
		return errors.New("slice field is required")
	}

	val := reflect.ValueOf(value)
	if val.Kind() != reflect.Slice {
		return errors.New("unexpected non-slice type")
	}

	if val.IsNil() || val.Len() == 0 {
		if sv.Optional {
			return nil
		}
		return errors.New("slice cannot be empty")
	}

	valLen := uint32(val.Len())
	if valLen < sv.MinLen {
		return fmt.Errorf("requires at least %d elements, it has %d", sv.MinLen, valLen)
	}
	if sv.MaxLen > 0 && valLen > sv.MaxLen {
		return fmt.Errorf("allows at most %d elements, it has %d", sv.MaxLen, valLen)
	}

	if sv.Validator == nil {
		return nil
	}

	// run validator through each slice element
	for i := 0; i < val.Len(); i++ {
		var err error

		elem := val.Index(i)
		if !elem.CanInterface() {
			return fmt.Errorf("unsupported element in slice")
		}

		kind := elem.Kind()
		if kind == reflect.Ptr {
			if elem.IsNil() {
				err = sv.Validator.Validate(nil)
			} else {
				err = sv.Validator.Validate(reflect.Indirect(elem).Interface())
			}
		} else {
			err = sv.Validator.Validate(elem.Interface())
		}

		if err != nil {
			return fmt.Errorf("index %d: %v", i, err)
		}
	}

	return nil
}

// ========== String Field Validator ========== //

type StringFunc func(string) error

type String struct {
	Optional   bool
	UnsetZero  bool
	MinLen     uint32
	MaxLen     uint32
	Charset    string
	Regex      *regexp.Regexp
	Validators []StringFunc
}

func (stv *String) Validate(value interface{}) error {
	if value == nil {
		if stv.Optional {
			return nil
		}
		return errors.New("string field is required")
	}

	str, ok := value.(string)
	if !ok {
		return errors.New("unexpected non-string type")
	}

	if str == "" {
		if stv.Optional {
			if stv.UnsetZero {
				return &defaultValue{}
			}
			return nil
		}
		return errors.New("string field cannot be empty")
	}

	runes := []rune(str)
	runesLen := uint32(len(runes))
	if runesLen < stv.MinLen {
		return fmt.Errorf("requires at least %d chars, it has %d", stv.MinLen, runesLen)
	}
	if stv.MaxLen > 0 && runesLen > stv.MaxLen {
		return fmt.Errorf("allows at most %d chars, it has %d", stv.MaxLen, runesLen)
	}

	if len(stv.Charset) > 0 {
		for i := 0; i < len(runes); i++ {
			if !strings.ContainsRune(stv.Charset, runes[i]) {
				return fmt.Errorf("must contain only chars '%v'", stv.Charset)
			}
		}
	}

	for _, v := range stv.Validators {
		if v != nil {
			if err := v(str); err != nil {
				return err
			}
		}
	}

	return nil
}

// ========== Bool Field Validator ========== //

type Bool struct {
	Optional bool
}

func (b *Bool) Validate(value interface{}) error {
	if value == nil {
		if b.Optional {
			return nil
		}
		return errors.New("bool field is required")
	}

	_, ok := value.(bool)
	if !ok {
		return errors.New("unexpected non-bool type")
	}

	return nil
}

// ========== UInt64 Field Validator ========== //

type UInt64Func func(uint64) error

type UInt64 struct {
	Optional   bool
	UnsetZero  bool
	Min        *uint64
	Max        *uint64
	Validators []UInt64Func
}

func (uv *UInt64) Validate(value interface{}) error {
	if value == nil {
		if uv.Optional {
			return nil
		}
		return errors.New("uint64 field is required")
	}

	ui, ok := value.(uint64)
	if !ok {
		return errors.New("unexpected non-uint64 type")
	}

	if ui == 0 {
		if uv.Optional {
			if uv.UnsetZero {
				return &defaultValue{}
			}
			return nil
		}
		return errors.New("uint64 field cannot be zero")
	}

	if uv.Min != nil && ui < *uv.Min {
		return fmt.Errorf("must be greater than or equal to %v", *uv.Min)
	}

	if uv.Max != nil && ui > *uv.Max {
		return fmt.Errorf("must be lesser than or equal to %v", *uv.Max)
	}

	for _, v := range uv.Validators {
		if v != nil {
			if err := v(ui); err != nil {
				return err
			}
		}
	}

	return nil
}

// ========== UInt32 Field Validator ========== //

type UInt32Func func(uint32) error

type UInt32 struct {
	Optional   bool
	UnsetZero  bool
	Min        *uint32
	Max        *uint32
	Validators []UInt32Func
}

func (uv *UInt32) Validate(value interface{}) error {
	if value == nil {
		if uv.Optional {
			return nil
		}
		return errors.New("uint32 field is required")
	}

	ui, ok := value.(uint32)
	if !ok {
		return errors.New("unexpected non-uint32 type")
	}

	if ui == 0 {
		if uv.Optional {
			if uv.UnsetZero {
				return &defaultValue{}
			}
			return nil
		}
		return errors.New("uint32 field cannot be zero")
	}

	if uv.Min != nil && ui < *uv.Min {
		return fmt.Errorf("must be greater than or equal to %v", *uv.Min)
	}

	if uv.Max != nil && ui > *uv.Max {
		return fmt.Errorf("must be lesser than or equal to %v", *uv.Max)
	}

	for _, v := range uv.Validators {
		if v != nil {
			if err := v(ui); err != nil {
				return err
			}
		}
	}

	return nil
}

// ========== Int64 Field Validator ========== //

type Int64Func func(int64) error

type Int64 struct {
	Optional   bool
	UnsetZero  bool
	Min        *int64
	Max        *int64
	Validators []Int64Func
}

func (uv *Int64) Validate(value interface{}) error {
	if value == nil {
		if uv.Optional {
			return nil
		}
		return errors.New("uint32 field is required")
	}

	ui, ok := value.(int64)
	if !ok {
		return errors.New("unexpected non-uint32 type")
	}

	if ui == 0 {
		if uv.Optional {
			if uv.UnsetZero {
				return &defaultValue{}
			}
			return nil
		}
		return errors.New("uint32 field cannot be zero")
	}

	if uv.Min != nil && ui < *uv.Min {
		return fmt.Errorf("must be greater than or equal to %v", *uv.Min)
	}

	if uv.Max != nil && ui > *uv.Max {
		return fmt.Errorf("must be lesser than or equal to %v", *uv.Max)
	}

	for _, v := range uv.Validators {
		if v != nil {
			if err := v(ui); err != nil {
				return err
			}
		}
	}

	return nil
}
