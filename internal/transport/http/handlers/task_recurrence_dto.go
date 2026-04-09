package handlers

import (
	"fmt"
	"strings"
	"time"

	taskrecurrencedomain "example.com/taskservice/internal/domain/taskrecurrence"
	taskrecurrenceusecase "example.com/taskservice/internal/usecase/taskrecurrence"
)

type taskRecurrenceMutationDTO struct {
	Title         string                    `json:"title"`
	Description   string                    `json:"description"`
	Type          taskrecurrencedomain.Type `json:"type"`
	StartsAt      time.Time                 `json:"starts_at"`
	EndsOn        *time.Time                `json:"ends_on,omitempty"`
	IntervalDays  *int                      `json:"interval_days,omitempty"`
	DayOfMonth    *int                      `json:"day_of_month,omitempty"`
	SpecificDates []string                  `json:"specific_dates,omitempty"`
}

type taskRecurrenceDTO struct {
	ID            int64                     `json:"id"`
	Title         string                    `json:"title"`
	Description   string                    `json:"description"`
	Type          taskrecurrencedomain.Type `json:"type"`
	StartsAt      time.Time                 `json:"starts_at"`
	EndsOn        *time.Time                `json:"ends_on,omitempty"`
	IntervalDays  *int                      `json:"interval_days,omitempty"`
	DayOfMonth    *int                      `json:"day_of_month,omitempty"`
	SpecificDates []string                  `json:"specific_dates,omitempty"`
	CreatedAt     time.Time                 `json:"created_at"`
	UpdatedAt     time.Time                 `json:"updated_at"`
}

func (dto taskRecurrenceMutationDTO) toCreateInput() (taskrecurrenceusecase.CreateInput, error) {
	specificDates, err := parseDateOnlyStrings(dto.SpecificDates)
	if err != nil {
		return taskrecurrenceusecase.CreateInput{}, err
	}

	return taskrecurrenceusecase.CreateInput{
		Title:         dto.Title,
		Description:   dto.Description,
		Type:          dto.Type,
		StartsAt:      dto.StartsAt,
		EndsOn:        dto.EndsOn,
		IntervalDays:  dto.IntervalDays,
		DayOfMonth:    dto.DayOfMonth,
		SpecificDates: specificDates,
	}, nil
}

func (dto taskRecurrenceMutationDTO) toUpdateInput() (taskrecurrenceusecase.UpdateInput, error) {
	specificDates, err := parseDateOnlyStrings(dto.SpecificDates)
	if err != nil {
		return taskrecurrenceusecase.UpdateInput{}, err
	}

	return taskrecurrenceusecase.UpdateInput{
		Title:         dto.Title,
		Description:   dto.Description,
		Type:          dto.Type,
		StartsAt:      dto.StartsAt,
		EndsOn:        dto.EndsOn,
		IntervalDays:  dto.IntervalDays,
		DayOfMonth:    dto.DayOfMonth,
		SpecificDates: specificDates,
	}, nil
}

func newTaskRecurrenceDTO(recurrence *taskrecurrencedomain.TaskRecurrence) taskRecurrenceDTO {
	return taskRecurrenceDTO{
		ID:            recurrence.ID,
		Title:         recurrence.Title,
		Description:   recurrence.Description,
		Type:          recurrence.Type,
		StartsAt:      recurrence.StartsAt,
		EndsOn:        recurrence.EndsOn,
		IntervalDays:  recurrence.IntervalDays,
		DayOfMonth:    recurrence.DayOfMonth,
		SpecificDates: formatDateOnlyStrings(recurrence.SpecificDates),
		CreatedAt:     recurrence.CreatedAt,
		UpdatedAt:     recurrence.UpdatedAt,
	}
}

func parseDateOnlyStrings(values []string) ([]time.Time, error) {
	if len(values) == 0 {
		return nil, nil
	}

	result := make([]time.Time, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return nil, fmt.Errorf("specific_dates must not contain empty values")
		}

		parsed, err := time.Parse("2006-01-02", trimmed)
		if err != nil {
			return nil, fmt.Errorf("specific_dates must contain valid date values in YYYY-MM-DD format")
		}

		result = append(result, parsed.UTC())
	}

	return result, nil
}

func formatDateOnlyStrings(values []time.Time) []string {
	if len(values) == 0 {
		return nil
	}

	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.UTC().Format("2006-01-02"))
	}

	return result
}
