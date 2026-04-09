package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "example.com/taskservice/internal/domain/task"
)

const taskColumns = `
	id,
	title,
	description,
	status,
	scheduled_at,
	recurrence_id,
	created_at,
	updated_at
`

type TaskRepository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *TaskRepository {
	return NewTaskRepository(pool)
}

func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{pool: pool}
}

func (r *TaskRepository) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		INSERT INTO tasks (
			title,
			description,
			status,
			scheduled_at,
			recurrence_id,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2,
			$3,
			COALESCE($4::timestamptz, NOW()),
			$5,
			COALESCE($6::timestamptz, NOW()),
			COALESCE($7::timestamptz, NOW())
		)
		RETURNING ` + taskColumns + `
	`

	row := r.pool.QueryRow(
		ctx,
		query,
		task.Title,
		task.Description,
		task.Status,
		nullableTimeValue(task.ScheduledAt),
		nullableInt64Value(task.RecurrenceID),
		nullableTimeValue(task.CreatedAt),
		nullableTimeValue(task.UpdatedAt),
	)
	created, err := scanTask(row)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *TaskRepository) CreateMany(ctx context.Context, tasks []taskdomain.Task) ([]taskdomain.Task, error) {
	if len(tasks) == 0 {
		return []taskdomain.Task{}, nil
	}

	const query = `
		INSERT INTO tasks (
			title,
			description,
			status,
			scheduled_at,
			recurrence_id,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2,
			$3,
			COALESCE($4::timestamptz, NOW()),
			$5,
			COALESCE($6::timestamptz, NOW()),
			COALESCE($7::timestamptz, NOW())
		)
		RETURNING ` + taskColumns + `
	`

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	batch := &pgx.Batch{}
	for i := range tasks {
		task := tasks[i]
		batch.Queue(
			query,
			task.Title,
			task.Description,
			task.Status,
			nullableTimeValue(task.ScheduledAt),
			nullableInt64Value(task.RecurrenceID),
			nullableTimeValue(task.CreatedAt),
			nullableTimeValue(task.UpdatedAt),
		)
	}

	results := tx.SendBatch(ctx, batch)

	created := make([]taskdomain.Task, 0, len(tasks))
	for range tasks {
		task, err := scanTask(results.QueryRow())
		if err != nil {
			_ = results.Close()
			return nil, err
		}

		created = append(created, *task)
	}

	if err := results.Close(); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return created, nil
}

func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	const query = `
		SELECT ` + taskColumns + `
		FROM tasks
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	found, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return found, nil
}

func (r *TaskRepository) Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		UPDATE tasks
		SET title = $1,
			description = $2,
			status = $3,
			scheduled_at = COALESCE($4::timestamptz, scheduled_at),
			recurrence_id = COALESCE($5::bigint, recurrence_id),
			updated_at = COALESCE($6::timestamptz, NOW())
		WHERE id = $7
		RETURNING ` + taskColumns + `
	`

	row := r.pool.QueryRow(
		ctx,
		query,
		task.Title,
		task.Description,
		task.Status,
		nullableTimeValue(task.ScheduledAt),
		nullableInt64Value(task.RecurrenceID),
		nullableTimeValue(task.UpdatedAt),
		task.ID,
	)
	updated, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return updated, nil
}

func (r *TaskRepository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM tasks WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}
	return nil
}

func (r *TaskRepository) DeleteFutureByRecurrenceID(ctx context.Context, recurrenceID int64, from time.Time) error {
	const query = `
		DELETE FROM tasks
		WHERE recurrence_id = $1
			AND scheduled_at >= $2
	`

	_, err := r.pool.Exec(ctx, query, recurrenceID, from)
	return err
}

func (r *TaskRepository) List(ctx context.Context) ([]taskdomain.Task, error) {
	const query = `
		SELECT ` + taskColumns + `
		FROM tasks
		ORDER BY id DESC
	`

	return r.listByQuery(ctx, query)
}

func (r *TaskRepository) ListByRecurrenceID(ctx context.Context, recurrenceID int64) ([]taskdomain.Task, error) {
	const query = `
		SELECT ` + taskColumns + `
		FROM tasks
		WHERE recurrence_id = $1
		ORDER BY scheduled_at ASC, id ASC
	`

	return r.listByQuery(ctx, query, recurrenceID)
}

func (r *TaskRepository) ListScheduledBetween(ctx context.Context, from, to time.Time) ([]taskdomain.Task, error) {
	const query = `
		SELECT ` + taskColumns + `
		FROM tasks
		WHERE scheduled_at >= $1
			AND scheduled_at <= $2
		ORDER BY scheduled_at ASC, id ASC
	`

	return r.listByQuery(ctx, query, from, to)
}

func (r *TaskRepository) listByQuery(ctx context.Context, query string, args ...any) ([]taskdomain.Task, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, *task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (*taskdomain.Task, error) {
	var (
		task         taskdomain.Task
		status       string
		recurrenceID sql.NullInt64
	)

	if err := scanner.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&status,
		&task.ScheduledAt,
		&recurrenceID,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return nil, err
	}

	task.Status = taskdomain.Status(status)
	if recurrenceID.Valid {
		id := recurrenceID.Int64
		task.RecurrenceID = &id
	}

	return &task, nil
}

func nullableTimeValue(value time.Time) any {
	if value.IsZero() {
		return nil
	}

	return value
}

func nullableInt64Value(value *int64) any {
	if value == nil {
		return nil
	}

	return *value
}
func nullableTimePointerValue(value *time.Time) any {
	if value == nil {
		return nil
	}

	return *value
}

func nullableIntValue(value *int) any {
	if value == nil {
		return nil
	}

	return *value
}
