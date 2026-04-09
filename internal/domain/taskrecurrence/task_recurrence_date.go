package taskrecurrence

import "time"

type TaskRecurrenceDate struct {
	RecurrenceID   int64     `json:"recurrence_id"`
	OccurrenceDate time.Time `json:"occurrence_date"`
}
