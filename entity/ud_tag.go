package entity

type UdTag struct {
	MappingID *MappingID `json:"mapping_id,omitempty"`
	Tag       *Tag       `json:"tag,omitempty"`
	TagValue  *string    `json:"tag_value,omitempty"`
}
