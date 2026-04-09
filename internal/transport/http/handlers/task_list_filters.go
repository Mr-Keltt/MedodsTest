package handlers

import (
	"fmt"
	"strings"
	"time"

	"net/http"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type taskListFilters struct {
	scheduledFrom *time.Time
	scheduledTo   *time.Time
	status        *taskdomain.Status
}

func parseTaskListFilters(r *http.Request) (taskListFilters, error) {
	query := r.URL.Query()
	filters := taskListFilters{}

	if rawScheduledFrom := strings.TrimSpace(query.Get("scheduled_from")); rawScheduledFrom != "" {
		parsed, err := time.Parse(time.RFC3339, rawScheduledFrom)
		if err != nil {
			return taskListFilters{}, fmt.Errorf("invalid scheduled_from")
		}

		filters.scheduledFrom = &parsed
	}

	if rawScheduledTo := strings.TrimSpace(query.Get("scheduled_to")); rawScheduledTo != "" {
		parsed, err := time.Parse(time.RFC3339, rawScheduledTo)
		if err != nil {
			return taskListFilters{}, fmt.Errorf("invalid scheduled_to")
		}

		filters.scheduledTo = &parsed
	}

	if filters.scheduledFrom != nil && filters.scheduledTo != nil && filters.scheduledTo.Before(*filters.scheduledFrom) {
		return taskListFilters{}, fmt.Errorf("scheduled_to must not be earlier than scheduled_from")
	}

	if rawStatus := strings.TrimSpace(query.Get("status")); rawStatus != "" {
		status := taskdomain.Status(rawStatus)
		if !status.Valid() {
			return taskListFilters{}, fmt.Errorf("invalid status")
		}

		filters.status = &status
	}

	return filters, nil
}

func (f taskListFilters) Matches(task taskdomain.Task) bool {
	if f.scheduledFrom != nil && task.ScheduledAt.Before(*f.scheduledFrom) {
		return false
	}

	if f.scheduledTo != nil && task.ScheduledAt.After(*f.scheduledTo) {
		return false
	}

	if f.status != nil && task.Status != *f.status {
		return false
	}

	return true
}
