package reply

import (
	"context"
	"errors"
	"sync"
	"time"

	"unwind/internal/plan"
)

var ErrBudgetExceeded = errors.New("daily model budget exceeded")

// DailyBudget limits model calls within one process and calendar day.
type DailyBudget struct {
	primary plan.Generator
	limit   int
	mu      sync.Mutex
	day     string
	count   int
	now     func() time.Time
}

func NewDailyBudget(primary plan.Generator, limit int) plan.Generator {
	return &DailyBudget{primary: primary, limit: limit, now: time.Now}
}

func (b *DailyBudget) Generate(ctx context.Context, day int, text string) (plan.Reply, error) {
	b.mu.Lock()
	today := b.now().Format("2006-01-02")
	if b.day != today {
		b.day = today
		b.count = 0
	}
	if b.count >= b.limit {
		b.mu.Unlock()
		return plan.Reply{}, ErrBudgetExceeded
	}
	b.count++
	b.mu.Unlock()
	return b.primary.Generate(ctx, day, text)
}
