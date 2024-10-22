package entity

type TaskAction uint32

const (
	TaskActionAdd TaskAction = iota
	TaskActionDelete
)

type ProfileType uint32

const (
	ProfileTypeUnknown ProfileType = iota
	ProfileTypeTag
)

type TaskStatus uint32

const (
	TaskStatusUnknown TagStatus = iota
	TaskStatusPending
	TaskStatusProcessing
	TaskStatusSuccess
	TaskStatusFailed
)

type Task struct {
	ID         *uint64 `json:"id,omitempty"`
	TagID      *uint64 `json:"tag_id,omitempty"`
	TagValue   *string `json:"tag_value,omitempty"`
	FileName   *string `json:"file_name,omitempty"`
	FileKey    *string `json:"file_key,omitempty"`
	URL        *string `json:"url,omitempty"`
	Status     *uint32 `json:"status,omitempty"`
	Action     *uint32 `json:"action,omitempty"`
	CreateTime *uint64 `json:"create_time,omitempty"`
	UpdateTime *uint64 `json:"update_time,omitempty"`
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

func (e *Task) GetAction() uint32 {
	if e != nil && e.Action != nil {
		return *e.Action
	}
	return 0
}

func (e *Task) GetStatus() uint32 {
	if e != nil && e.Status != nil {
		return *e.Status
	}
	return 0
}

func (e *Task) IsPending() bool {
	return e.GetStatus() == uint32(TaskStatusPending)
}

func (e *Task) IsAdd() bool {
	return e.GetAction() == uint32(TaskActionAdd)
}

func (e *Task) IsDelete() bool {
	return e.GetAction() == uint32(TaskActionDelete)
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
