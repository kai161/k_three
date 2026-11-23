package plan

import (
	"context"
	"sort"
	"strings"
	"unicode/utf8"
)

// Service coordinates entitlement, sequential progress and daily responses.
type Service struct {
	store      Store
	generator  Generator
	publicBeta bool
}

func NewService(store Store, generator Generator) *Service {
	return NewServiceWithOptions(store, generator, Options{})
}

func NewServiceWithOptions(store Store, generator Generator, options Options) *Service {
	return &Service{store: store, generator: generator, publicBeta: options.PublicBeta}
}

func (s *Service) Current(ctx context.Context, deviceID string) (Status, error) {
	progress, err := s.store.Progress(ctx, deviceID)
	if err != nil {
		return Status{}, err
	}
	return statusFrom(progress, s.publicBeta), nil
}

func (s *Service) Generate(ctx context.Context, deviceID, text string) (Reply, error) {
	text = strings.TrimSpace(text)
	if text == "" || utf8.RuneCountInString(text) > 200 {
		return Reply{}, ErrInvalidInput
	}
	status, err := s.Current(ctx, deviceID)
	if err != nil {
		return Reply{}, err
	}
	if status.Finished || status.NextDay > TotalDays {
		return Reply{}, ErrInvalidDay
	}
	if status.NextDay > FreeDays && !status.Activated && !s.publicBeta {
		return Reply{}, ErrLocked
	}
	return s.generator.Generate(ctx, status.NextDay, text)
}

func (s *Service) Complete(ctx context.Context, deviceID string, day int, mood, ritual string) (Status, error) {
	progress, err := s.store.Progress(ctx, deviceID)
	if err != nil {
		return Status{}, err
	}
	status := statusFrom(progress, s.publicBeta)
	if day < 1 || day > TotalDays || day > status.NextDay {
		return Status{}, ErrInvalidDay
	}
	if day > FreeDays && !status.Activated && !s.publicBeta {
		return Status{}, ErrLocked
	}
	if day < status.NextDay {
		return status, nil
	}
	entry, err := s.store.Complete(ctx, deviceID, day, strings.TrimSpace(mood), strings.TrimSpace(ritual))
	if err != nil {
		return Status{}, err
	}
	progress.Entries = append(progress.Entries, entry)
	return statusFrom(progress, s.publicBeta), nil
}

func (s *Service) Activate(ctx context.Context, deviceID, code string) (Status, error) {
	if err := s.store.Activate(ctx, deviceID, strings.TrimSpace(code)); err != nil {
		return Status{}, err
	}
	return s.Current(ctx, deviceID)
}

func (s *Service) SaveFeedback(ctx context.Context, deviceID string, day int, text string) error {
	text = strings.TrimSpace(text)
	if (day != 2 && day != 7) || text == "" || utf8.RuneCountInString(text) > 500 {
		return ErrInvalidInput
	}
	status, err := s.Current(ctx, deviceID)
	if err != nil {
		return err
	}
	if status.CompletedDays < day {
		return ErrInvalidInput
	}
	feedbackStore, ok := s.store.(FeedbackStore)
	if !ok {
		return ErrFeedbackUnsupported
	}
	return feedbackStore.SaveFeedback(ctx, deviceID, day, text)
}

func (s *Service) Review(ctx context.Context, deviceID string) (Review, error) {
	status, err := s.Current(ctx, deviceID)
	if err != nil {
		return Review{}, err
	}
	if !status.Finished {
		return Review{}, ErrNotFinished
	}
	counts := make(map[string]int)
	for _, entry := range status.Entries {
		counts[entry.Mood]++
	}
	moods := make([]string, 0, len(counts))
	for mood := range counts {
		moods = append(moods, mood)
	}
	sort.Strings(moods)
	mostFrequent := ""
	for _, mood := range moods {
		if counts[mood] > counts[mostFrequent] {
			mostFrequent = mood
		}
	}
	return Review{
		CompletedDays:    TotalDays,
		MostFrequentMood: mostFrequent,
		Message:          "这七天，你为情绪留出了一点位置。不需要每天都变好，愿意停下来看看自己，就已经足够。",
	}, nil
}

func statusFrom(progress Progress, publicBeta bool) Status {
	completed := len(progress.Entries)
	next := completed + 1
	if next > TotalDays {
		next = TotalDays
	}
	return Status{
		Activated:     progress.Activated,
		PublicBeta:    publicBeta,
		CompletedDays: completed,
		NextDay:       next,
		Finished:      completed >= TotalDays,
		FreeDays:      FreeDays,
		TotalDays:     TotalDays,
		Entries:       progress.Entries,
	}
}
