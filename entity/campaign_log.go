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

type LogExtra struct {
	Link    *string `json:"link,omitempty"`
	TsEpoch *uint64 `json:"ts_epoch,omitempty"`
	Date    *string `json:"date,omitempty"`
	Email   *string `json:"email,omitempty"`
}

type CampaignLog struct {
	ID              *uint64  `json:"id,omitempty"`
	CampaignEmailID *uint64  `json:"campaign_email_id,omitempty"`
	Event           Event    `json:"event,omitempty"`
	LogExtra        LogExtra `json:"log_extra,omitempty"`
	CreateTime      *uint64  `json:"create_time,omitempty"`
}

func (e *CampaignLog) GetEvent() Event {
	if e != nil {
		return e.Event
	}
	return EventUnknown
}
