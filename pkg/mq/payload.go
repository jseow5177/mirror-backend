package mq

type Payload uint32

const (
	PayloadUnknown Payload = iota
	PayloadNotifyCreateTask
)

var Payloads = map[Payload]string{
	PayloadNotifyCreateTask: "notify_create_task",
}

type NotifyCreateTask struct {
	TaskID *uint64 `json:"task_id"`
}

func (m *NotifyCreateTask) GetTaskID() uint64 {
	if m != nil && m.TaskID != nil {
		return *m.TaskID
	}
	return 0
}
