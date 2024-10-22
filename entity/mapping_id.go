package entity

type MappingID struct {
	ID   *uint64 `json:"id,omitempty"`
	UdID *string `json:"ud_id,omitempty"`
}

func (e *MappingID) GetID() uint64 {
	if e != nil && e.ID != nil {
		return *e.ID
	}
	return 0
}

func (e *MappingID) GetUdID() string {
	if e != nil && e.UdID != nil {
		return *e.UdID
	}
	return ""
}
