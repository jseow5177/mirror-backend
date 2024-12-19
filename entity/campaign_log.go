package entity

type Event uint32

const (
	EventUnknown Event = iota
	EventUniqueOpened
	EventClick
)

var SupportedEvents = map[string]Event{
	"unique_opened": EventUniqueOpened,
	"click":         EventClick,
}

type CampaignLog struct {
	ID              *uint64 `json:"id,omitempty"`
	CampaignEmailID *uint64 `json:"campaign_email_id,omitempty"`
	Event           Event   `json:"event,omitempty"`
	Link            *string `json:"link,omitempty"`
	Email           *string `json:"email,omitempty"`
	EventTime       *uint64 `json:"event_time,omitempty"`
	CreateTime      *uint64 `json:"create_time,omitempty"`
}

func (e *CampaignLog) GetEvent() Event {
	if e != nil {
		return e.Event
	}
	return EventUnknown
}
