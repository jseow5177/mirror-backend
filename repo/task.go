package repo

import (
	"cdp/config"
	"cdp/entity"
	"context"
	"errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	ErrTaskNotFound = errors.New("task not found")
)

// fields that can be updated
var mFields = []string{"status"}

type Task struct {
	ID         *uint64
	TagID      *uint64
	TagValue   *string
	FileName   *string
	FileKey    *string
	URL        *string
	Status     *uint32
	Action     *uint32
	CreateTime *uint64
	UpdateTime *uint64
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

type TaskFilter struct {
	ID *uint64
}

type TaskRepo interface {
	Create(ctx context.Context, task *entity.Task) (uint64, error)
	Update(_ context.Context, f *TaskFilter, task *entity.Task) error
	Get(ctx context.Context, f *TaskFilter) (*entity.Task, error)
	Close(ctx context.Context) error
}

type taskRepo struct {
	orm *gorm.DB
}

func NewTaskRepo(_ context.Context, mysqlCfg config.MySQL) (TaskRepo, error) {
	orm, err := gorm.Open(mysql.Open(mysqlCfg.ToDSN()))
	if err != nil {
		return nil, err
	}

	return &taskRepo{
		orm: orm,
	}, nil
}

func (r *taskRepo) Get(_ context.Context, f *TaskFilter) (*entity.Task, error) {
	task := new(Task)
	if err := r.orm.Where(f).First(task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	return ToTask(task), nil
}

func (r *taskRepo) Update(_ context.Context, f *TaskFilter, task *entity.Task) error {
	return r.orm.Where(f).Select(mFields).Updates(ToTaskModel(task)).Error
}

func (r *taskRepo) Create(_ context.Context, task *entity.Task) (uint64, error) {
	taskModel := ToTaskModel(task)

	if err := r.orm.Create(taskModel).Error; err != nil {
		return 0, err
	}

	return taskModel.GetID(), nil
}

func (r *taskRepo) Close(_ context.Context) error {
	if r.orm != nil {
		sqlDB, err := r.orm.DB()
		if err != nil {
			return err
		}

		err = sqlDB.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func ToTaskModel(task *entity.Task) *Task {
	return &Task{
		ID:         task.ID,
		TagID:      task.TagID,
		TagValue:   task.TagValue,
		FileName:   task.FileName,
		FileKey:    task.FileKey,
		URL:        task.URL,
		Status:     task.Status,
		Action:     task.Action,
		CreateTime: task.CreateTime,
		UpdateTime: task.UpdateTime,
	}
}

func ToTask(task *Task) *entity.Task {
	return &entity.Task{
		ID:         task.ID,
		TagID:      task.TagID,
		TagValue:   task.TagValue,
		FileName:   task.FileName,
		FileKey:    task.FileKey,
		URL:        task.URL,
		Status:     task.Status,
		Action:     task.Action,
		CreateTime: task.CreateTime,
		UpdateTime: task.UpdateTime,
	}
}
