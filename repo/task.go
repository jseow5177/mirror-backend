package repo

import (
	"cdp/entity"
	"cdp/pkg/goutil"
	"context"
	"encoding/json"
)

type Task struct {
	ID           *uint64
	TenantID     *uint64
	ResourceID   *uint64
	Status       *uint32
	ResourceType *uint32
	TaskType     *uint32
	ExtInfo      *string
	CreatorID    *uint64
	CreateTime   *uint64
	UpdateTime   *uint64
}

func (m *Task) TableName() string {
	return "task_tab"
}

func (m *Task) GetID() uint64 {
	if m != nil && m.ID != nil {
		return *m.ID
	}
	return 0
}

func (m *Task) GetStatus() uint32 {
	if m != nil && m.Status != nil {
		return *m.Status
	}
	return 0
}

func (m *Task) GetTaskType() uint32 {
	if m != nil && m.TaskType != nil {
		return *m.TaskType
	}
	return 0
}

func (m *Task) GetResourceType() uint32 {
	if m != nil && m.ResourceType != nil {
		return *m.ResourceType
	}
	return 0
}

type TaskRepo interface {
	Create(ctx context.Context, task *entity.Task) (uint64, error)
	Update(ctx context.Context, task *entity.Task) error
	GetByResourceIDAndType(ctx context.Context, resourceID uint64, resourceType entity.ResourceType, p *Pagination) ([]*entity.Task, *Pagination, error)
	GetPendingFileUploadTasks(ctx context.Context, resourceType entity.ResourceType) ([]*entity.Task, error)
}

func NewTaskRepo(_ context.Context, baseRepo BaseRepo) TaskRepo {
	return &taskRepo{
		baseRepo: baseRepo,
	}
}

type taskRepo struct {
	baseRepo BaseRepo
}

func (r *taskRepo) GetPendingFileUploadTasks(ctx context.Context, resourceType entity.ResourceType) ([]*entity.Task, error) {
	tasks, _, err := r.getMany(ctx, []*Condition{
		{
			Field:         "resource_type",
			Value:         resourceType,
			Op:            OpEq,
			NextLogicalOp: LogicalOpAnd,
		},
		{
			Field:         "task_type",
			Value:         entity.TaskTypeFileUpload,
			Op:            OpEq,
			NextLogicalOp: LogicalOpAnd,
		},
		{
			Field: "status",
			Value: entity.TaskStatusPending,
			Op:    OpEq,
		},
	}, nil)
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *taskRepo) GetByResourceIDAndType(ctx context.Context, resourceID uint64, resourceType entity.ResourceType, p *Pagination) ([]*entity.Task, *Pagination, error) {
	return r.getMany(ctx, []*Condition{
		{
			Field:         "resource_id",
			Value:         resourceID,
			Op:            OpEq,
			NextLogicalOp: LogicalOpAnd,
		},
		{
			Field: "resource_type",
			Value: resourceType,
			Op:    OpEq,
		},
	}, p)
}

func (r *taskRepo) Update(ctx context.Context, task *entity.Task) error {
	taskModel, err := ToTaskModel(task)
	if err != nil {
		return err
	}

	if err := r.baseRepo.Update(ctx, taskModel); err != nil {
		return err
	}

	return nil
}

func (r *taskRepo) getMany(ctx context.Context, conditions []*Condition, p *Pagination) ([]*entity.Task, *Pagination, error) {
	res, pNew, err := r.baseRepo.GetMany(ctx, new(Task), &Filter{
		Conditions: conditions,
		Pagination: p,
	})
	if err != nil {
		return nil, nil, err
	}

	tasks := make([]*entity.Task, 0, len(res))
	for _, m := range res {
		task, err := ToTask(m.(*Task))
		if err != nil {
			return nil, nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, pNew, nil
}

func (r *taskRepo) Create(ctx context.Context, task *entity.Task) (uint64, error) {
	taskModel, err := ToTaskModel(task)
	if err != nil {
		return 0, err
	}

	if err := r.baseRepo.Create(ctx, taskModel); err != nil {
		return 0, err
	}

	return taskModel.GetID(), nil
}

func ToTaskModel(task *entity.Task) (*Task, error) {
	extInfo, err := task.GetExtInfo().ToString()
	if err != nil {
		return nil, err
	}

	return &Task{
		ID:           task.ID,
		TenantID:     task.TenantID,
		ResourceID:   task.ResourceID,
		Status:       goutil.Uint32(uint32(task.GetStatus())),
		TaskType:     goutil.Uint32(uint32(task.GetTaskType())),
		ResourceType: goutil.Uint32(uint32(task.GetResourceType())),
		ExtInfo:      goutil.String(extInfo),
		CreatorID:    task.CreatorID,
		CreateTime:   task.CreateTime,
		UpdateTime:   task.UpdateTime,
	}, nil
}

func ToTask(task *Task) (*entity.Task, error) {
	extInfo := new(entity.TaskExtInfo)
	if err := json.Unmarshal([]byte(*task.ExtInfo), extInfo); err != nil {
		return nil, err
	}

	return &entity.Task{
		ID:           task.ID,
		TenantID:     task.TenantID,
		ResourceID:   task.ResourceID,
		Status:       entity.TaskStatus(task.GetStatus()),
		TaskType:     entity.TaskType(task.GetTaskType()),
		ResourceType: entity.ResourceType(task.GetResourceType()),
		ExtInfo:      extInfo,
		CreatorID:    task.CreatorID,
		CreateTime:   task.CreateTime,
		UpdateTime:   task.UpdateTime,
	}, nil
}
