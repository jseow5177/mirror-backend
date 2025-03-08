package entity

import (
	"cdp/pkg/goutil"
	"encoding/json"
	"mime/multipart"
	"time"
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

func (e *TaskExtInfo) GetProgress() uint64 {
	if e != nil && e.Progress != nil {
		return *e.Progress
	}
	return 0
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
	TenantID     *uint64      `json:"tenant_id,omitempty"`
	ResourceID   *uint64      `json:"resource_id,omitempty"`
	Status       TaskStatus   `json:"status,omitempty"`
	TaskType     TaskType     `json:"task_type,omitempty"`
	ResourceType ResourceType `json:"resource_type,omitempty"`
	ExtInfo      *TaskExtInfo `json:"ext_info,omitempty"`
	CreatorID    *uint64      `json:"creator_id,omitempty"`
	CreateTime   *uint64      `json:"create_time,omitempty"`
	UpdateTime   *uint64      `json:"update_time,omitempty"`
}

func (e *Task) GetID() uint64 {
	if e != nil && e.ID != nil {
		return *e.ID
	}
	return 0
}

func (e *Task) GetTenantID() uint64 {
	if e != nil && e.TenantID != nil {
		return *e.TenantID
	}
	return 0
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

func (e *Task) GetResourceID() uint64 {
	if e != nil && e.ResourceID != nil {
		return *e.ResourceID
	}
	return 0
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

func (e *Task) GetFileID() string {
	if e != nil && e.ExtInfo != nil {
		return *e.ExtInfo.FileID
	}
	return ""
}

func (e *Task) GetSize() uint64 {
	if e != nil && e.ExtInfo != nil {
		return *e.ExtInfo.Size
	}
	return 0
}

func (e *Task) Update(newTask *Task) bool {
	var hasChange bool

	if newTask.Status != TaskStatusUnknown && e.Status != newTask.Status {
		hasChange = true
		e.Status = newTask.Status
	}

	if newTask.ExtInfo != nil {
		oldExtInfo := e.ExtInfo
		if oldExtInfo == nil {
			oldExtInfo = new(TaskExtInfo)
		}

		if newTask.ExtInfo.Progress != nil && oldExtInfo.GetProgress() != newTask.ExtInfo.GetProgress() {
			hasChange = true
			oldExtInfo.Progress = newTask.ExtInfo.Progress
		}
	}

	if hasChange {
		e.UpdateTime = goutil.Uint64(uint64(time.Now().Unix()))
	}

	return hasChange
}
