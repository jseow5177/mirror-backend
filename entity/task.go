package entity

import (
	"encoding/json"
	"errors"
	"mime/multipart"
)

var (
	ErrInvalidResourceType = errors.New("invalid resource type")
)

type FileMeta struct {
	File       multipart.File
	FileHeader *multipart.FileHeader
}

type ResourceType uint32

const (
	ResourceTypeUnknown ResourceType = iota
	ResourceTypeTag
)

var ResourceTypes = map[ResourceType]string{
	ResourceTypeTag: "tag",
}

func CheckResourceType(rt uint32) error {
	if _, ok := ResourceTypes[ResourceType(rt)]; !ok {
		return ErrInvalidResourceType
	}
	return nil
}

type TaskType uint32

const (
	TaskTypeUnknown TaskType = iota
	TaskTypeFileUpload
)

type TaskStatus uint32

const (
	TaskStatusUnknown TaskStatus = iota
	TaskStatusPending
	TaskStatusRunning
	TaskStatusSuccess
	TaskStatusFailed
)

type TaskExtInfo struct {
	FileID      *string `json:"file_id,omitempty"`
	OriFileName *string `json:"ori_file_name,omitempty"`
	Size        *uint64 `json:"size,omitempty"`
	Progress    *uint64 `json:"progress,omitempty"`
}

func (e *TaskExtInfo) ToString() (string, error) {
	if e == nil {
		return "{}", nil
	}

	b, err := json.Marshal(e)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

type Task struct {
	ID           *uint64      `json:"id,omitempty"`
	ResourceID   *uint64      `json:"resource_id,omitempty"`
	Status       TaskStatus   `json:"status,omitempty"`
	TaskType     TaskType     `json:"task_type,omitempty"`
	ResourceType ResourceType `json:"resource_type,omitempty"`
	ExtInfo      *TaskExtInfo `json:"ext_info,omitempty"`
	CreatorID    *uint64      `json:"creator_id,omitempty"`
	CreateTime   *uint64      `json:"create_time,omitempty"`
	UpdateTime   *uint64      `json:"update_time,omitempty"`
}

func (e *Task) GetTaskType() TaskType {
	if e != nil {
		return e.TaskType
	}
	return TaskTypeUnknown
}

func (e *Task) GetStatus() TaskStatus {
	if e != nil {
		return e.Status
	}
	return TaskStatusUnknown
}

func (e *Task) GetResourceType() ResourceType {
	if e != nil {
		return e.ResourceType
	}
	return ResourceTypeUnknown
}

func (e *Task) GetExtInfo() *TaskExtInfo {
	if e != nil && e.ExtInfo != nil {
		return e.ExtInfo
	}
	return nil
}
