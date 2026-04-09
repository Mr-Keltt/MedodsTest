ALTER TABLE tasks
ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMPTZ;

UPDATE tasks
SET scheduled_at = created_at
WHERE scheduled_at IS NULL;

ALTER TABLE tasks
ALTER COLUMN scheduled_at SET NOT NULL;

ALTER TABLE tasks
ALTER COLUMN scheduled_at SET DEFAULT NOW();

CREATE TABLE IF NOT EXISTS task_recurrences (
	id BIGSERIAL PRIMARY KEY,
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	type TEXT NOT NULL,
	starts_at TIMESTAMPTZ NOT NULL,
	ends_on TIMESTAMPTZ NULL,
	interval_days INTEGER NULL,
	day_of_month INTEGER NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CONSTRAINT chk_task_recurrences_type
		CHECK (type IN (
			'daily_every_n_days',
			'monthly_day_of_month',
			'specific_dates',
			'odd_days_of_month',
			'even_days_of_month'
		)),
	CONSTRAINT chk_task_recurrences_ends_on
		CHECK (ends_on IS NULL OR ends_on >= starts_at),
	CONSTRAINT chk_task_recurrences_interval_days
		CHECK (interval_days IS NULL OR interval_days >= 1),
	CONSTRAINT chk_task_recurrences_day_of_month
		CHECK (day_of_month IS NULL OR day_of_month BETWEEN 1 AND 30),
	CONSTRAINT chk_task_recurrences_type_fields
		CHECK (
			(type = 'daily_every_n_days' AND interval_days IS NOT NULL AND day_of_month IS NULL)
			OR
			(type = 'monthly_day_of_month' AND day_of_month IS NOT NULL AND interval_days IS NULL)
			OR
			(type = 'specific_dates' AND interval_days IS NULL AND day_of_month IS NULL)
			OR
			(type = 'odd_days_of_month' AND interval_days IS NULL AND day_of_month IS NULL)
			OR
			(type = 'even_days_of_month' AND interval_days IS NULL AND day_of_month IS NULL)
		)
);

ALTER TABLE tasks
ADD COLUMN IF NOT EXISTS recurrence_id BIGINT NULL;

DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1
		FROM pg_constraint
		WHERE conname = 'fk_tasks_recurrence_id'
	) THEN
		ALTER TABLE tasks
		ADD CONSTRAINT fk_tasks_recurrence_id
			FOREIGN KEY (recurrence_id)
			REFERENCES task_recurrences(id)
			ON DELETE SET NULL;
	END IF;
END $$;

CREATE TABLE IF NOT EXISTS task_recurrence_dates (
	recurrence_id BIGINT NOT NULL REFERENCES task_recurrences(id) ON DELETE CASCADE,
	occurrence_date DATE NOT NULL,
	PRIMARY KEY (recurrence_id, occurrence_date)
);

CREATE INDEX IF NOT EXISTS idx_tasks_scheduled_at ON tasks (scheduled_at);
CREATE INDEX IF NOT EXISTS idx_tasks_recurrence_id ON tasks (recurrence_id);
CREATE UNIQUE INDEX IF NOT EXISTS ux_tasks_recurrence_id_scheduled_at ON tasks (recurrence_id, scheduled_at) WHERE recurrence_id IS NOT NULL;