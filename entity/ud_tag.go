package entity

type IDType uint32

const (
	IDTypeUnknown IDType = iota
	IDTypeEmail
)

var IDTypes = map[IDType]string{
	IDTypeEmail: "email",
}

type Ud struct {
	ID     *string `json:"id,omitempty"`
	IDType IDType  `json:"id_type,omitempty"`
}

func (e *Ud) GetID() string {
	if e != nil && e.ID != nil {
		return *e.ID
	}
	return ""
}

type TagVal struct {
	TagID   *uint64 `json:"tag_id,omitempty"`
	TagName *string `json:"tag_name,omitempty"`
	TagVal  *string `json:"tag_val,omitempty"`
}

type UdTagVal struct {
	Ud      *Ud       `json:"ud,omitempty"`
	TagVals []*TagVal `json:"tag_vals,omitempty"`
}

type UdTags struct {
	Tag  *Tag     `json:"tag,omitempty"`
	Data []*UdTag `json:"data,omitempty"`
}

type UdTag struct {
	MappingID *MappingID `json:"mapping_id,omitempty"`
	TagValue  *string    `json:"tag_value,omitempty"`
}
