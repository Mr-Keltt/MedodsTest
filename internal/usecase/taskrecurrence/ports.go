package taskrecurrence

import (
	"context"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	taskrecurrencedomain "example.com/taskservice/internal/domain/taskrecurrence"
)

type Repository interface {
	Create(ctx context.Context, recurrence *taskrecurrencedomain.TaskRecurrence) (*taskrecurrencedomain.TaskRecurrence, error)
	GetByID(ctx context.Context, id int64) (*taskrecurrencedomain.TaskRecurrence, error)
	Update(ctx context.Context, recurrence *taskrecurrencedomain.TaskRecurrence) (*taskrecurrencedomain.TaskRecurrence, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]taskrecurrencedomain.TaskRecurrence, error)
	ReplaceDates(ctx context.Context, recurrenceID int64, dates []time.Time) error
	ListDates(ctx context.Context, recurrenceID int64) ([]time.Time, error)
}

type TaskRepository interface {
	CreateMany(ctx context.Context, tasks []taskdomain.Task) ([]taskdomain.Task, error)
	DeleteFutureByRecurrenceID(ctx context.Context, recurrenceID int64, from time.Time) error
	ListByRecurrenceID(ctx context.Context, recurrenceID int64) ([]taskdomain.Task, error)
}

type Usecase interface {
	Create(ctx context.Context, input CreateInput) (*taskrecurrencedomain.TaskRecurrence, error)
	GetByID(ctx context.Context, id int64) (*taskrecurrencedomain.TaskRecurrence, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*taskrecurrencedomain.TaskRecurrence, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]taskrecurrencedomain.TaskRecurrence, error)
	SyncFutureTasks(ctx context.Context) error
}

type CreateInput struct {
	Title         string
	Description   string
	Type          taskrecurrencedomain.Type
	StartsAt      time.Time
	EndsOn        *time.Time
	IntervalDays  *int
	DayOfMonth    *int
	SpecificDates []time.Time
}

type UpdateInput struct {
	Title         string
	Description   string
	Type          taskrecurrencedomain.Type
	StartsAt      time.Time
	EndsOn        *time.Time
	IntervalDays  *int
	DayOfMonth    *int
	SpecificDates []time.Time
}
