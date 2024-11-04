package entity

type Pagination struct {
	Page    *uint32 `json:"page,omitempty"`
	Limit   *uint32 `json:"limit,omitempty"`
	HasNext *bool   `json:"has_next,omitempty"`
	Total   *int64  `json:"total,omitempty"`
}

func (e *Pagination) GetPage() uint32 {
	if e != nil && e.Page != nil {
		return *e.Page
	}
	return 0
}

func (e *Pagination) GetLimit() uint32 {
	if e != nil && e.Limit != nil {
		return *e.Limit
	}
	return 10
}

func (e *Pagination) GetHasNext() bool {
	if e != nil && e.HasNext != nil {
		return *e.HasNext
	}
	return false
}

func (e *Pagination) GetTotal() int64 {
	if e != nil && e.Total != nil {
		return *e.Total
	}
	return 0
}
