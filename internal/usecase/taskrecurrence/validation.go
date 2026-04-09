package taskrecurrence

import (
	"fmt"
	"slices"
	"strings"
	"time"

	taskrecurrencedomain "example.com/taskservice/internal/domain/taskrecurrence"
)

func validateCreateInput(input CreateInput) (CreateInput, error) {
	return validateMutationInput(input.Title, input.Description, input.Type, input.StartsAt, input.EndsOn, input.IntervalDays, input.DayOfMonth, input.SpecificDates)
}

func validateUpdateInput(input UpdateInput) (UpdateInput, error) {
	validated, err := validateMutationInput(input.Title, input.Description, input.Type, input.StartsAt, input.EndsOn, input.IntervalDays, input.DayOfMonth, input.SpecificDates)
	if err != nil {
		return UpdateInput{}, err
	}

	return UpdateInput(validated), nil
}

func validateMutationInput(
	title string,
	description string,
	recurrenceType taskrecurrencedomain.Type,
	startsAt time.Time,
	endsOn *time.Time,
	intervalDays *int,
	dayOfMonth *int,
	specificDates []time.Time,
) (CreateInput, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)

	if title == "" {
		return CreateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if startsAt.IsZero() {
		return CreateInput{}, fmt.Errorf("%w: starts_at is required", ErrInvalidInput)
	}

	if !recurrenceType.Valid() {
		return CreateInput{}, fmt.Errorf("%w: invalid recurrence type", ErrInvalidInput)
	}

	if endsOn != nil && endsOn.Before(startsAt) {
		return CreateInput{}, fmt.Errorf("%w: ends_on must not be earlier than starts_at", ErrInvalidInput)
	}

	normalizedDates, err := normalizeSpecificDates(specificDates, startsAt.Location())
	if err != nil {
		return CreateInput{}, err
	}

	switch recurrenceType {
	case taskrecurrencedomain.TypeDailyEveryNDays:
		if intervalDays == nil {
			return CreateInput{}, fmt.Errorf("%w: interval_days is required for daily recurrence", ErrInvalidInput)
		}
		if *intervalDays < 1 {
			return CreateInput{}, fmt.Errorf("%w: interval_days must be greater than or equal to 1", ErrInvalidInput)
		}
		if dayOfMonth != nil {
			return CreateInput{}, fmt.Errorf("%w: day_of_month is not allowed for daily recurrence", ErrInvalidInput)
		}
		if len(normalizedDates) > 0 {
			return CreateInput{}, fmt.Errorf("%w: specific_dates is not allowed for daily recurrence", ErrInvalidInput)
		}
	case taskrecurrencedomain.TypeMonthlyDayOfMonth:
		if dayOfMonth == nil {
			return CreateInput{}, fmt.Errorf("%w: day_of_month is required for monthly recurrence", ErrInvalidInput)
		}
		if *dayOfMonth < 1 || *dayOfMonth > 30 {
			return CreateInput{}, fmt.Errorf("%w: day_of_month must be between 1 and 30", ErrInvalidInput)
		}
		if intervalDays != nil {
			return CreateInput{}, fmt.Errorf("%w: interval_days is not allowed for monthly recurrence", ErrInvalidInput)
		}
		if len(normalizedDates) > 0 {
			return CreateInput{}, fmt.Errorf("%w: specific_dates is not allowed for monthly recurrence", ErrInvalidInput)
		}
	case taskrecurrencedomain.TypeSpecificDates:
		if len(normalizedDates) == 0 {
			return CreateInput{}, fmt.Errorf("%w: specific_dates must not be empty for specific_dates recurrence", ErrInvalidInput)
		}
		if intervalDays != nil {
			return CreateInput{}, fmt.Errorf("%w: interval_days is not allowed for specific_dates recurrence", ErrInvalidInput)
		}
		if dayOfMonth != nil {
			return CreateInput{}, fmt.Errorf("%w: day_of_month is not allowed for specific_dates recurrence", ErrInvalidInput)
		}
	case taskrecurrencedomain.TypeOddDaysOfMonth, taskrecurrencedomain.TypeEvenDaysOfMonth:
		if intervalDays != nil {
			return CreateInput{}, fmt.Errorf("%w: interval_days is not allowed for odd/even day recurrence", ErrInvalidInput)
		}
		if dayOfMonth != nil {
			return CreateInput{}, fmt.Errorf("%w: day_of_month is not allowed for odd/even day recurrence", ErrInvalidInput)
		}
		if len(normalizedDates) > 0 {
			return CreateInput{}, fmt.Errorf("%w: specific_dates is not allowed for odd/even day recurrence", ErrInvalidInput)
		}
	default:
		return CreateInput{}, fmt.Errorf("%w: unsupported recurrence type", ErrInvalidInput)
	}

	return CreateInput{
		Title:         title,
		Description:   description,
		Type:          recurrenceType,
		StartsAt:      startsAt,
		EndsOn:        endsOn,
		IntervalDays:  cloneIntPointer(intervalDays),
		DayOfMonth:    cloneIntPointer(dayOfMonth),
		SpecificDates: normalizedDates,
	}, nil
}

func normalizeSpecificDates(dates []time.Time, location *time.Location) ([]time.Time, error) {
	if len(dates) == 0 {
		return nil, nil
	}

	normalized := make([]time.Time, 0, len(dates))
	seen := make(map[string]struct{}, len(dates))

	for _, date := range dates {
		if date.IsZero() {
			return nil, fmt.Errorf("%w: specific_dates must not contain zero values", ErrInvalidInput)
		}

		normalizedDate := normalizeDateOnly(date, location)
		key := dateOnlyKey(normalizedDate)

		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("%w: specific_dates must not contain duplicates", ErrInvalidInput)
		}

		seen[key] = struct{}{}
		normalized = append(normalized, normalizedDate)
	}

	slices.SortFunc(normalized, func(a, b time.Time) int {
		switch {
		case a.Before(b):
			return -1
		case a.After(b):
			return 1
		default:
			return 0
		}
	})

	return normalized, nil
}

func normalizeDateOnly(value time.Time, location *time.Location) time.Time {
	if location == nil {
		location = time.UTC
	}

	local := value.In(location)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}

func dateOnlyKey(value time.Time) string {
	return value.Format("2006-01-02")
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func cloneDates(values []time.Time) []time.Time {
	if len(values) == 0 {
		return nil
	}

	cloned := make([]time.Time, len(values))
	copy(cloned, values)
	return cloned
}
