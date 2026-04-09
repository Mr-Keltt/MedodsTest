package taskrecurrence

import "time"

type Type string

const (
	TypeDailyEveryNDays   Type = "daily_every_n_days"
	TypeMonthlyDayOfMonth Type = "monthly_day_of_month"
	TypeSpecificDates     Type = "specific_dates"
	TypeOddDaysOfMonth    Type = "odd_days_of_month"
	TypeEvenDaysOfMonth   Type = "even_days_of_month"
)

type TaskRecurrence struct {
	ID           int64      `json:"id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	Type         Type       `json:"type"`
	StartsAt     time.Time  `json:"starts_at"`
	EndsOn       *time.Time `json:"ends_on,omitempty"`
	IntervalDays *int       `json:"interval_days,omitempty"`
	DayOfMonth   *int       `json:"day_of_month,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (t Type) Valid() bool {
	switch t {
	case TypeDailyEveryNDays,
		TypeMonthlyDayOfMonth,
		TypeSpecificDates,
		TypeOddDaysOfMonth,
		TypeEvenDaysOfMonth:
		return true
	default:
		return false
	}
}

func ValidTypes() []Type {
	return []Type{
		TypeDailyEveryNDays,
		TypeMonthlyDayOfMonth,
		TypeSpecificDates,
		TypeOddDaysOfMonth,
		TypeEvenDaysOfMonth,
	}
}
