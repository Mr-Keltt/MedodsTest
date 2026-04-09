package taskrecurrence

import (
	"fmt"
	"sort"
	"time"

	taskrecurrencedomain "example.com/taskservice/internal/domain/taskrecurrence"
)

func GenerateScheduledTimes(
	recurrence taskrecurrencedomain.TaskRecurrence,
	from time.Time,
	to time.Time,
	specificDates []time.Time,
) ([]time.Time, error) {
	if !recurrence.Type.Valid() {
		return nil, fmt.Errorf("%w: invalid recurrence type", taskrecurrencedomain.ErrInvalidType)
	}

	effectiveFrom, effectiveTo, ok := clampGenerationRange(recurrence, from, to)
	if !ok {
		return []time.Time{}, nil
	}

	var occurrences []time.Time

	switch recurrence.Type {
	case taskrecurrencedomain.TypeDailyEveryNDays:
		occurrences = generateDailyOccurrences(recurrence, effectiveFrom, effectiveTo)
	case taskrecurrencedomain.TypeMonthlyDayOfMonth:
		occurrences = generateMonthlyOccurrences(recurrence, effectiveFrom, effectiveTo)
	case taskrecurrencedomain.TypeSpecificDates:
		occurrences = generateSpecificDateOccurrences(recurrence, effectiveFrom, effectiveTo, specificDates)
	case taskrecurrencedomain.TypeOddDaysOfMonth:
		occurrences = generateOddEvenOccurrences(recurrence, effectiveFrom, effectiveTo, true)
	case taskrecurrencedomain.TypeEvenDaysOfMonth:
		occurrences = generateOddEvenOccurrences(recurrence, effectiveFrom, effectiveTo, false)
	default:
		return nil, fmt.Errorf("%w: unsupported recurrence type", taskrecurrencedomain.ErrInvalidType)
	}

	sort.Slice(occurrences, func(i, j int) bool {
		return occurrences[i].Before(occurrences[j])
	})

	return occurrences, nil
}

func clampGenerationRange(
	recurrence taskrecurrencedomain.TaskRecurrence,
	from time.Time,
	to time.Time,
) (time.Time, time.Time, bool) {
	if to.Before(from) {
		return time.Time{}, time.Time{}, false
	}

	effectiveFrom := from
	if recurrence.StartsAt.After(effectiveFrom) {
		effectiveFrom = recurrence.StartsAt
	}

	effectiveTo := to
	if recurrence.EndsOn != nil && recurrence.EndsOn.Before(effectiveTo) {
		effectiveTo = *recurrence.EndsOn
	}

	if effectiveTo.Before(effectiveFrom) {
		return time.Time{}, time.Time{}, false
	}

	return effectiveFrom, effectiveTo, true
}

func generateDailyOccurrences(
	recurrence taskrecurrencedomain.TaskRecurrence,
	from time.Time,
	to time.Time,
) []time.Time {
	intervalDays := *recurrence.IntervalDays
	current := recurrence.StartsAt

	for current.Before(from) {
		current = current.AddDate(0, 0, intervalDays)
	}

	occurrences := make([]time.Time, 0)
	for !current.After(to) {
		occurrences = append(occurrences, current)
		current = current.AddDate(0, 0, intervalDays)
	}

	return occurrences
}

func generateMonthlyOccurrences(
	recurrence taskrecurrencedomain.TaskRecurrence,
	from time.Time,
	to time.Time,
) []time.Time {
	location := recurrence.StartsAt.Location()
	dayOfMonth := *recurrence.DayOfMonth
	occurrences := make([]time.Time, 0)

	currentMonth := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, location)
	lastMonth := time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, location)

	for !currentMonth.After(lastMonth) {
		candidate := time.Date(
			currentMonth.Year(),
			currentMonth.Month(),
			dayOfMonth,
			recurrence.StartsAt.Hour(),
			recurrence.StartsAt.Minute(),
			recurrence.StartsAt.Second(),
			recurrence.StartsAt.Nanosecond(),
			location,
		)

		if candidate.Month() == currentMonth.Month() &&
			!candidate.Before(recurrence.StartsAt) &&
			!candidate.Before(from) &&
			!candidate.After(to) {
			occurrences = append(occurrences, candidate)
		}

		currentMonth = currentMonth.AddDate(0, 1, 0)
	}

	return occurrences
}

func generateSpecificDateOccurrences(
	recurrence taskrecurrencedomain.TaskRecurrence,
	from time.Time,
	to time.Time,
	specificDates []time.Time,
) []time.Time {
	location := recurrence.StartsAt.Location()
	occurrences := make([]time.Time, 0, len(specificDates))

	for _, date := range specificDates {
		candidate := time.Date(
			date.Year(),
			date.Month(),
			date.Day(),
			recurrence.StartsAt.Hour(),
			recurrence.StartsAt.Minute(),
			recurrence.StartsAt.Second(),
			recurrence.StartsAt.Nanosecond(),
			location,
		)

		if candidate.Before(recurrence.StartsAt) || candidate.Before(from) || candidate.After(to) {
			continue
		}

		occurrences = append(occurrences, candidate)
	}

	return occurrences
}

func generateOddEvenOccurrences(
	recurrence taskrecurrencedomain.TaskRecurrence,
	from time.Time,
	to time.Time,
	odd bool,
) []time.Time {
	location := recurrence.StartsAt.Location()
	currentDate := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, location)
	lastDate := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, location)
	occurrences := make([]time.Time, 0)

	for !currentDate.After(lastDate) {
		candidate := time.Date(
			currentDate.Year(),
			currentDate.Month(),
			currentDate.Day(),
			recurrence.StartsAt.Hour(),
			recurrence.StartsAt.Minute(),
			recurrence.StartsAt.Second(),
			recurrence.StartsAt.Nanosecond(),
			location,
		)

		isOddDay := currentDate.Day()%2 == 1
		if isOddDay == odd && !candidate.Before(recurrence.StartsAt) && !candidate.Before(from) && !candidate.After(to) {
			occurrences = append(occurrences, candidate)
		}

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return occurrences
}
