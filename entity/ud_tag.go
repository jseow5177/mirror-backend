package entity

type UdTags struct {
	Tag  *Tag     `json:"tag,omitempty"`
	Data []*UdTag `json:"data,omitempty"`
}

type UdTag struct {
	MappingID *MappingID `json:"mapping_id,omitempty"`
	TagValue  *string    `json:"tag_value,omitempty"`
}
