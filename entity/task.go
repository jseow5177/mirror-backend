package entity

type TaskStatus uint32

const (
	TaskStatusUnknown TaskStatus = iota
	TaskStatusPending
	TaskStatusProcessing
	TaskStatusSuccess
	TaskStatusFailed
)

type Task struct {
	ID         *uint64    `json:"id,omitempty"`
	TagID      *uint64    `json:"tag_id,omitempty"`
	FileName   *string    `json:"file_name,omitempty"`
	FileKey    *string    `json:"file_key,omitempty"`
	URL        *string    `json:"url,omitempty"`
	Status     TaskStatus `json:"status,omitempty"`
	CreateTime *uint64    `json:"create_time,omitempty"`
	UpdateTime *uint64    `json:"update_time,omitempty"`
}

func (e *Task) GetID() uint64 {
	if e != nil && e.ID != nil {
		return *e.ID
	}
	return 0
}

func (e *Task) GetTagID() uint64 {
	if e != nil && e.TagID != nil {
		return *e.TagID
	}
	return 0
}

func (e *Task) GetStatus() TaskStatus {
	if e != nil {
		return e.Status
	}
	return TaskStatusUnknown
}

func (e *Task) IsPending() bool {
	return e.GetStatus() == TaskStatusPending
}

func (e *Task) GetFileKey() string {
	if e != nil && e.FileKey != nil {
		return *e.FileKey
	}
	return ""
}

func (e *Task) GetFileName() string {
	if e != nil && e.FileName != nil {
		return *e.FileName
	}
	return ""
}
