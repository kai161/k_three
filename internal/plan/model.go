package plan

import (
	"context"
	"errors"
	"time"
)

const (
	TotalDays = 7
	FreeDays  = 2
)

var (
	ErrLocked              = errors.New("plan is locked")
	ErrInvalidDay          = errors.New("invalid plan day")
	ErrInvalidInput        = errors.New("invalid input")
	ErrNotFinished         = errors.New("plan is not finished")
	ErrFeedbackUnsupported = errors.New("feedback store is not supported")
)

// Entry is one completed daily ritual. It intentionally excludes the original reflection.
type Entry struct {
	Day         int       `json:"day"`
	Mood        string    `json:"mood"`
	Ritual      string    `json:"ritual"`
	CompletedAt time.Time `json:"completedAt"`
}

// Progress contains the entitlement and completed entries for a device.
type Progress struct {
	DeviceID  string  `json:"-"`
	Activated bool    `json:"activated"`
	Entries   []Entry `json:"entries"`
}

// Reply is a generated or curated response to one reflection.
type Reply struct {
	Message string `json:"message"`
	Action  string `json:"action"`
	Source  string `json:"source"`
}

// Options controls product-mode behavior without changing stored progress.
type Options struct {
	PublicBeta bool
}

// Status describes the user's current position in the seven-day plan.
type Status struct {
	Activated     bool    `json:"activated"`
	PublicBeta    bool    `json:"publicBeta"`
	CompletedDays int     `json:"completedDays"`
	NextDay       int     `json:"nextDay"`
	Finished      bool    `json:"finished"`
	FreeDays      int     `json:"freeDays"`
	TotalDays     int     `json:"totalDays"`
	Entries       []Entry `json:"entries"`
}

// Review is a non-diagnostic summary of completed rituals.
type Review struct {
	CompletedDays    int    `json:"completedDays"`
	MostFrequentMood string `json:"mostFrequentMood"`
	Message          string `json:"message"`
}

// Store persists plan state without storing raw reflections.
type Store interface {
	EnsureDevice(ctx context.Context, deviceID string) error
	Progress(ctx context.Context, deviceID string) (Progress, error)
	Complete(ctx context.Context, deviceID string, day int, mood, ritual string) (Entry, error)
	Activate(ctx context.Context, deviceID, code string) error
}

// FeedbackStore persists optional beta feedback independently of reflections.
type FeedbackStore interface {
	SaveFeedback(ctx context.Context, deviceID string, day int, text string) error
}

// Generator creates a response without persisting the supplied reflection.
type Generator interface {
	Generate(ctx context.Context, day int, text string) (Reply, error)
}
