package plan

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStore struct {
	progress Progress
	entry    Entry
}

func (f *fakeStore) EnsureDevice(context.Context, string) error { return nil }
func (f *fakeStore) Progress(context.Context, string) (Progress, error) {
	return f.progress, nil
}
func (f *fakeStore) Complete(_ context.Context, _ string, day int, mood, ritual string) (Entry, error) {
	f.entry = Entry{Day: day, Mood: mood, Ritual: ritual, CompletedAt: time.Now()}
	return f.entry, nil
}
func (f *fakeStore) Activate(context.Context, string, string) error { return nil }

type fakeGenerator struct {
	reply Reply
	err   error
	text  string
}

func (f *fakeGenerator) Generate(_ context.Context, _ int, text string) (Reply, error) {
	f.text = text
	return f.reply, f.err
}

func TestServiceGenerateFreeAndLockedDays(t *testing.T) {
	generator := &fakeGenerator{reply: Reply{Message: "辛苦了", Action: "喝口水", Source: "ai"}}
	store := &fakeStore{}
	service := NewService(store, generator)

	if _, err := service.Generate(context.Background(), "device", "今天开了很多会"); err != nil {
		t.Fatalf("first day should be free: %v", err)
	}
	store.progress.Entries = []Entry{{Day: 1}, {Day: 2}}
	if _, err := service.Generate(context.Background(), "device", "还在想工作"); !errors.Is(err, ErrLocked) {
		t.Fatalf("third day without activation = %v, want ErrLocked", err)
	}
	store.progress.Activated = true
	if _, err := service.Generate(context.Background(), "device", "还在想工作"); err != nil {
		t.Fatalf("third day after activation: %v", err)
	}
}

func TestServiceCompleteIsSequentialAndIdempotent(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, &fakeGenerator{})

	status, err := service.Complete(context.Background(), "device", 1, "平静", "揉掉")
	if err != nil {
		t.Fatal(err)
	}
	if status.CompletedDays != 1 || status.NextDay != 2 {
		t.Fatalf("unexpected status: %+v", status)
	}

	status, err = service.Complete(context.Background(), "device", 1, "平静", "揉掉")
	if err != nil || status.CompletedDays != 1 {
		t.Fatalf("repeat completion should be idempotent: status=%+v err=%v", status, err)
	}

	if _, err = service.Complete(context.Background(), "device", 3, "平静", "揉掉"); !errors.Is(err, ErrInvalidDay) {
		t.Fatalf("skipping a day = %v, want ErrInvalidDay", err)
	}
}

func TestServiceReviewRequiresSevenCompletedDays(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, &fakeGenerator{})
	if _, err := service.Review(context.Background(), "device"); !errors.Is(err, ErrNotFinished) {
		t.Fatalf("review before day seven = %v, want ErrNotFinished", err)
	}

	for day := 1; day <= 7; day++ {
		store.progress.Entries = append(store.progress.Entries, Entry{Day: day, Mood: "平静"})
	}
	review, err := service.Review(context.Background(), "device")
	if err != nil {
		t.Fatal(err)
	}
	if review.CompletedDays != 7 || review.MostFrequentMood != "平静" {
		t.Fatalf("unexpected review: %+v", review)
	}
}

func TestServiceRejectsEmptyAndOverlongInput(t *testing.T) {
	service := NewService(&fakeStore{}, &fakeGenerator{})
	if _, err := service.Generate(context.Background(), "device", "  "); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("empty input = %v, want ErrInvalidInput", err)
	}
	overlong := make([]rune, 201)
	for i := range overlong {
		overlong[i] = '事'
	}
	if _, err := service.Generate(context.Background(), "device", string(overlong)); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("overlong input = %v, want ErrInvalidInput", err)
	}
}

func TestPublicBetaAllowsAllDaysWithoutActivation(t *testing.T) {
	generator := &fakeGenerator{reply: Reply{Message: "辛苦了", Action: "喝口水", Source: "ai"}}
	store := &fakeStore{progress: Progress{Entries: []Entry{{Day: 1}, {Day: 2}}}}
	service := NewServiceWithOptions(store, generator, Options{PublicBeta: true})

	status, err := service.Current(context.Background(), "device")
	if err != nil || !status.PublicBeta {
		t.Fatalf("public beta status = %+v, err=%v", status, err)
	}
	if _, err := service.Generate(context.Background(), "device", "第三天仍然免费"); err != nil {
		t.Fatalf("public beta day three: %v", err)
	}
	if _, err := service.Complete(context.Background(), "device", 3, "平静", "揉掉"); err != nil {
		t.Fatalf("public beta completion: %v", err)
	}
}

func TestServiceRejectsFeedbackBeforeItsDay(t *testing.T) {
	service := NewService(&fakeStore{}, &fakeGenerator{})
	if err := service.SaveFeedback(context.Background(), "device", 1, "太早了"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("day one feedback error = %v, want ErrInvalidInput", err)
	}
	if err := service.SaveFeedback(context.Background(), "device", 2, " "); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("empty feedback error = %v, want ErrInvalidInput", err)
	}
	if err := service.SaveFeedback(context.Background(), "device", 7, "尚未完成"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("incomplete feedback error = %v, want ErrInvalidInput", err)
	}
}
