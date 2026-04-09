package taskrecurrence

import (
	"testing"
	"time"

	taskrecurrencedomain "example.com/taskservice/internal/domain/taskrecurrence"
)

func TestGenerateScheduledTimes_DailyEveryNDays(t *testing.T) {
	recurrence := taskrecurrencedomain.TaskRecurrence{
		Type:         taskrecurrencedomain.TypeDailyEveryNDays,
		StartsAt:     time.Date(2026, time.April, 1, 9, 0, 0, 0, time.UTC),
		IntervalDays: intPtr(2),
	}

	from := time.Date(2026, time.April, 3, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.April, 8, 23, 59, 59, 0, time.UTC)

	actual, err := GenerateScheduledTimes(recurrence, from, to, nil)
	if err != nil {
		t.Fatalf("GenerateScheduledTimes returned error: %v", err)
	}

	expected := []time.Time{
		time.Date(2026, time.April, 3, 9, 0, 0, 0, time.UTC),
		time.Date(2026, time.April, 5, 9, 0, 0, 0, time.UTC),
		time.Date(2026, time.April, 7, 9, 0, 0, 0, time.UTC),
	}

	assertTimesEqual(t, expected, actual)
}

func TestGenerateScheduledTimes_MonthlyDayOfMonthSkipsInvalidMonth(t *testing.T) {
	recurrence := taskrecurrencedomain.TaskRecurrence{
		Type:       taskrecurrencedomain.TypeMonthlyDayOfMonth,
		StartsAt:   time.Date(2026, time.January, 10, 8, 30, 0, 0, time.UTC),
		DayOfMonth: intPtr(30),
	}

	from := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.March, 31, 23, 59, 59, 0, time.UTC)

	actual, err := GenerateScheduledTimes(recurrence, from, to, nil)
	if err != nil {
		t.Fatalf("GenerateScheduledTimes returned error: %v", err)
	}

	expected := []time.Time{
		time.Date(2026, time.March, 30, 8, 30, 0, 0, time.UTC),
	}

	assertTimesEqual(t, expected, actual)
}

func TestGenerateScheduledTimes_SpecificDates(t *testing.T) {
	recurrence := taskrecurrencedomain.TaskRecurrence{
		Type:     taskrecurrencedomain.TypeSpecificDates,
		StartsAt: time.Date(2026, time.April, 10, 14, 15, 0, 0, time.UTC),
	}

	specificDates := []time.Time{
		time.Date(2026, time.April, 9, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.April, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC),
	}

	from := time.Date(2026, time.April, 10, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.April, 20, 23, 59, 59, 0, time.UTC)

	actual, err := GenerateScheduledTimes(recurrence, from, to, specificDates)
	if err != nil {
		t.Fatalf("GenerateScheduledTimes returned error: %v", err)
	}

	expected := []time.Time{
		time.Date(2026, time.April, 10, 14, 15, 0, 0, time.UTC),
		time.Date(2026, time.April, 15, 14, 15, 0, 0, time.UTC),
	}

	assertTimesEqual(t, expected, actual)
}

func TestGenerateScheduledTimes_OddDaysOfMonth(t *testing.T) {
	recurrence := taskrecurrencedomain.TaskRecurrence{
		Type:     taskrecurrencedomain.TypeOddDaysOfMonth,
		StartsAt: time.Date(2026, time.April, 1, 7, 0, 0, 0, time.UTC),
	}

	from := time.Date(2026, time.April, 2, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.April, 7, 23, 59, 59, 0, time.UTC)

	actual, err := GenerateScheduledTimes(recurrence, from, to, nil)
	if err != nil {
		t.Fatalf("GenerateScheduledTimes returned error: %v", err)
	}

	expected := []time.Time{
		time.Date(2026, time.April, 3, 7, 0, 0, 0, time.UTC),
		time.Date(2026, time.April, 5, 7, 0, 0, 0, time.UTC),
		time.Date(2026, time.April, 7, 7, 0, 0, 0, time.UTC),
	}

	assertTimesEqual(t, expected, actual)
}

func TestGenerateScheduledTimes_EvenDaysOfMonth(t *testing.T) {
	recurrence := taskrecurrencedomain.TaskRecurrence{
		Type:     taskrecurrencedomain.TypeEvenDaysOfMonth,
		StartsAt: time.Date(2026, time.April, 1, 7, 0, 0, 0, time.UTC),
	}

	from := time.Date(2026, time.April, 2, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.April, 7, 23, 59, 59, 0, time.UTC)

	actual, err := GenerateScheduledTimes(recurrence, from, to, nil)
	if err != nil {
		t.Fatalf("GenerateScheduledTimes returned error: %v", err)
	}

	expected := []time.Time{
		time.Date(2026, time.April, 2, 7, 0, 0, 0, time.UTC),
		time.Date(2026, time.April, 4, 7, 0, 0, 0, time.UTC),
		time.Date(2026, time.April, 6, 7, 0, 0, 0, time.UTC),
	}

	assertTimesEqual(t, expected, actual)
}

func intPtr(value int) *int {
	return &value
}

func assertTimesEqual(t *testing.T, expected, actual []time.Time) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Fatalf("length mismatch: expected %d, got %d", len(expected), len(actual))
	}

	for i := range expected {
		if !expected[i].Equal(actual[i]) {
			t.Fatalf("element %d mismatch: expected %s, got %s", i, expected[i], actual[i])
		}
	}
}
