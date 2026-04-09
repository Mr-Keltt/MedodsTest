package taskrecurrence

import (
	"context"
	"fmt"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	taskrecurrencedomain "example.com/taskservice/internal/domain/taskrecurrence"
)

const defaultPlanningHorizonDays = 60

type Service struct {
	repo                Repository
	taskRepo            TaskRepository
	now                 func() time.Time
	planningHorizonDays int
}

func NewService(repo Repository, taskRepo TaskRepository) *Service {
	return &Service{
		repo:                repo,
		taskRepo:            taskRepo,
		now:                 func() time.Time { return time.Now().UTC() },
		planningHorizonDays: defaultPlanningHorizonDays,
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*taskrecurrencedomain.TaskRecurrence, error) {
	normalized, err := validateCreateInput(input)
	if err != nil {
		return nil, err
	}

	now := s.now()
	model := &taskrecurrencedomain.TaskRecurrence{
		Title:         normalized.Title,
		Description:   normalized.Description,
		Type:          normalized.Type,
		StartsAt:      normalized.StartsAt,
		EndsOn:        cloneTimePointer(normalized.EndsOn),
		IntervalDays:  cloneIntPointer(normalized.IntervalDays),
		DayOfMonth:    cloneIntPointer(normalized.DayOfMonth),
		SpecificDates: cloneDates(normalized.SpecificDates),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	created, err := s.repo.Create(ctx, model)
	if err != nil {
		return nil, err
	}

	if err := s.replaceSpecificDatesIfNeeded(ctx, created.ID, created.Type, normalized.SpecificDates); err != nil {
		return nil, err
	}

	created.SpecificDates = cloneDates(normalized.SpecificDates)

	if err := s.materializeRecurrence(ctx, created, now, s.horizonEnd(now)); err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*taskrecurrencedomain.TaskRecurrence, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	recurrence, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.attachSpecificDates(ctx, recurrence); err != nil {
		return nil, err
	}

	return recurrence, nil
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*taskrecurrencedomain.TaskRecurrence, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	normalized, err := validateUpdateInput(input)
	if err != nil {
		return nil, err
	}

	now := s.now()
	model := &taskrecurrencedomain.TaskRecurrence{
		ID:            id,
		Title:         normalized.Title,
		Description:   normalized.Description,
		Type:          normalized.Type,
		StartsAt:      normalized.StartsAt,
		EndsOn:        cloneTimePointer(normalized.EndsOn),
		IntervalDays:  cloneIntPointer(normalized.IntervalDays),
		DayOfMonth:    cloneIntPointer(normalized.DayOfMonth),
		SpecificDates: cloneDates(normalized.SpecificDates),
		UpdatedAt:     now,
	}

	updated, err := s.repo.Update(ctx, model)
	if err != nil {
		return nil, err
	}

	if err := s.replaceSpecificDatesIfNeeded(ctx, updated.ID, updated.Type, normalized.SpecificDates); err != nil {
		return nil, err
	}

	if err := s.taskRepo.DeleteFutureByRecurrenceID(ctx, updated.ID, now); err != nil {
		return nil, err
	}

	updated.SpecificDates = cloneDates(normalized.SpecificDates)

	if err := s.materializeRecurrence(ctx, updated, now, s.horizonEnd(now)); err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	now := s.now()
	if err := s.taskRepo.DeleteFutureByRecurrenceID(ctx, id, now); err != nil {
		return err
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]taskrecurrencedomain.TaskRecurrence, error) {
	recurrences, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	for i := range recurrences {
		if err := s.attachSpecificDates(ctx, &recurrences[i]); err != nil {
			return nil, err
		}
	}

	return recurrences, nil
}

func (s *Service) SyncFutureTasks(ctx context.Context) error {
	now := s.now()
	recurrences, err := s.List(ctx)
	if err != nil {
		return err
	}

	horizonEnd := s.horizonEnd(now)
	for i := range recurrences {
		if err := s.materializeRecurrence(ctx, &recurrences[i], now, horizonEnd); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) materializeRecurrence(
	ctx context.Context,
	recurrence *taskrecurrencedomain.TaskRecurrence,
	from time.Time,
	to time.Time,
) error {
	occurrences, err := GenerateScheduledTimes(*recurrence, from, to, recurrence.SpecificDates)
	if err != nil {
		return err
	}

	if len(occurrences) == 0 {
		return nil
	}

	existingTasks, err := s.taskRepo.ListByRecurrenceID(ctx, recurrence.ID)
	if err != nil {
		return err
	}

	existingByScheduledAt := make(map[string]struct{}, len(existingTasks))
	for _, task := range existingTasks {
		existingByScheduledAt[task.ScheduledAt.UTC().Format(time.RFC3339Nano)] = struct{}{}
	}

	now := s.now()
	recurrenceID := recurrence.ID
	tasksToCreate := make([]taskdomain.Task, 0)

	for _, occurrence := range occurrences {
		key := occurrence.UTC().Format(time.RFC3339Nano)
		if _, exists := existingByScheduledAt[key]; exists {
			continue
		}

		tasksToCreate = append(tasksToCreate, taskdomain.Task{
			Title:        recurrence.Title,
			Description:  recurrence.Description,
			Status:       taskdomain.StatusNew,
			ScheduledAt:  occurrence,
			RecurrenceID: &recurrenceID,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}

	if len(tasksToCreate) == 0 {
		return nil
	}

	_, err = s.taskRepo.CreateMany(ctx, tasksToCreate)
	return err
}

func (s *Service) attachSpecificDates(ctx context.Context, recurrence *taskrecurrencedomain.TaskRecurrence) error {
	if recurrence.Type != taskrecurrencedomain.TypeSpecificDates {
		recurrence.SpecificDates = nil
		return nil
	}

	dates, err := s.repo.ListDates(ctx, recurrence.ID)
	if err != nil {
		return err
	}

	recurrence.SpecificDates = cloneDates(dates)
	return nil
}

func (s *Service) replaceSpecificDatesIfNeeded(
	ctx context.Context,
	recurrenceID int64,
	recurrenceType taskrecurrencedomain.Type,
	dates []time.Time,
) error {
	if recurrenceType == taskrecurrencedomain.TypeSpecificDates {
		return s.repo.ReplaceDates(ctx, recurrenceID, cloneDates(dates))
	}

	return s.repo.ReplaceDates(ctx, recurrenceID, nil)
}

func (s *Service) horizonEnd(now time.Time) time.Time {
	return now.AddDate(0, 0, s.planningHorizonDays)
}
