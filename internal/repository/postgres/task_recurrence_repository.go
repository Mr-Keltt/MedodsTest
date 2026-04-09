package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	taskrecurrencedomain "example.com/taskservice/internal/domain/taskrecurrence"
)

const taskRecurrenceColumns = `
	id,
	title,
	description,
	type,
	starts_at,
	ends_on,
	interval_days,
	day_of_month,
	created_at,
	updated_at
`

type TaskRecurrenceRepository struct {
	pool *pgxpool.Pool
}

func NewTaskRecurrenceRepository(pool *pgxpool.Pool) *TaskRecurrenceRepository {
	return &TaskRecurrenceRepository{pool: pool}
}

func (r *TaskRecurrenceRepository) Create(
	ctx context.Context,
	recurrence *taskrecurrencedomain.TaskRecurrence,
) (*taskrecurrencedomain.TaskRecurrence, error) {
	const query = `
		INSERT INTO task_recurrences (
			title,
			description,
			type,
			starts_at,
			ends_on,
			interval_days,
			day_of_month,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			COALESCE($8::timestamptz, NOW()),
			COALESCE($9::timestamptz, NOW())
		)
		RETURNING ` + taskRecurrenceColumns + `
	`

	row := r.pool.QueryRow(
		ctx,
		query,
		recurrence.Title,
		recurrence.Description,
		recurrence.Type,
		nullableTimeValue(recurrence.StartsAt),
		nullableTimePointerValue(recurrence.EndsOn),
		nullableIntValue(recurrence.IntervalDays),
		nullableIntValue(recurrence.DayOfMonth),
		nullableTimeValue(recurrence.CreatedAt),
		nullableTimeValue(recurrence.UpdatedAt),
	)

	created, err := scanTaskRecurrence(row)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *TaskRecurrenceRepository) GetByID(
	ctx context.Context,
	id int64,
) (*taskrecurrencedomain.TaskRecurrence, error) {
	const query = `
		SELECT ` + taskRecurrenceColumns + `
		FROM task_recurrences
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	recurrence, err := scanTaskRecurrence(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskrecurrencedomain.ErrNotFound
		}

		return nil, err
	}

	return recurrence, nil
}

func (r *TaskRecurrenceRepository) Update(
	ctx context.Context,
	recurrence *taskrecurrencedomain.TaskRecurrence,
) (*taskrecurrencedomain.TaskRecurrence, error) {
	const query = `
		UPDATE task_recurrences
		SET title = $1,
			description = $2,
			type = $3,
			starts_at = $4,
			ends_on = $5,
			interval_days = $6,
			day_of_month = $7,
			updated_at = COALESCE($8::timestamptz, NOW())
		WHERE id = $9
		RETURNING ` + taskRecurrenceColumns + `
	`

	row := r.pool.QueryRow(
		ctx,
		query,
		recurrence.Title,
		recurrence.Description,
		recurrence.Type,
		nullableTimeValue(recurrence.StartsAt),
		nullableTimePointerValue(recurrence.EndsOn),
		nullableIntValue(recurrence.IntervalDays),
		nullableIntValue(recurrence.DayOfMonth),
		nullableTimeValue(recurrence.UpdatedAt),
		recurrence.ID,
	)

	updated, err := scanTaskRecurrence(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskrecurrencedomain.ErrNotFound
		}

		return nil, err
	}

	return updated, nil
}

func (r *TaskRecurrenceRepository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM task_recurrences WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return taskrecurrencedomain.ErrNotFound
	}

	return nil
}

func (r *TaskRecurrenceRepository) List(
	ctx context.Context,
) ([]taskrecurrencedomain.TaskRecurrence, error) {
	const query = `
		SELECT ` + taskRecurrenceColumns + `
		FROM task_recurrences
		ORDER BY id DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	recurrences := make([]taskrecurrencedomain.TaskRecurrence, 0)
	for rows.Next() {
		recurrence, err := scanTaskRecurrence(rows)
		if err != nil {
			return nil, err
		}

		recurrences = append(recurrences, *recurrence)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return recurrences, nil
}

func (r *TaskRecurrenceRepository) ReplaceDates(
	ctx context.Context,
	recurrenceID int64,
	dates []time.Time,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(
		ctx,
		`DELETE FROM task_recurrence_dates WHERE recurrence_id = $1`,
		recurrenceID,
	); err != nil {
		return err
	}

	if len(dates) == 0 {
		return tx.Commit(ctx)
	}

	batch := &pgx.Batch{}
	for _, date := range dates {
		batch.Queue(
			`INSERT INTO task_recurrence_dates (recurrence_id, occurrence_date) VALUES ($1, $2)`,
			recurrenceID,
			date,
		)
	}

	results := tx.SendBatch(ctx, batch)
	for range dates {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return err
		}
	}

	if err := results.Close(); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *TaskRecurrenceRepository) ListDates(ctx context.Context, recurrenceID int64) ([]time.Time, error) {
	const query = `
		SELECT occurrence_date
		FROM task_recurrence_dates
		WHERE recurrence_id = $1
		ORDER BY occurrence_date ASC
	`

	rows, err := r.pool.Query(ctx, query, recurrenceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dates := make([]time.Time, 0)
	for rows.Next() {
		var date time.Time
		if err := rows.Scan(&date); err != nil {
			return nil, err
		}

		dates = append(dates, date)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return dates, nil
}

type taskRecurrenceScanner interface {
	Scan(dest ...any) error
}

func scanTaskRecurrence(scanner taskRecurrenceScanner) (*taskrecurrencedomain.TaskRecurrence, error) {
	var (
		recurrence     taskrecurrencedomain.TaskRecurrence
		endsOn         sql.NullTime
		intervalDays   sql.NullInt32
		dayOfMonth     sql.NullInt32
		recurrenceType string
	)

	if err := scanner.Scan(
		&recurrence.ID,
		&recurrence.Title,
		&recurrence.Description,
		&recurrenceType,
		&recurrence.StartsAt,
		&endsOn,
		&intervalDays,
		&dayOfMonth,
		&recurrence.CreatedAt,
		&recurrence.UpdatedAt,
	); err != nil {
		return nil, err
	}

	recurrence.Type = taskrecurrencedomain.Type(recurrenceType)

	if endsOn.Valid {
		value := endsOn.Time
		recurrence.EndsOn = &value
	}

	if intervalDays.Valid {
		value := int(intervalDays.Int32)
		recurrence.IntervalDays = &value
	}

	if dayOfMonth.Valid {
		value := int(dayOfMonth.Int32)
		recurrence.DayOfMonth = &value
	}

	return &recurrence, nil
}
