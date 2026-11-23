package reply

import (
	"context"
	"errors"
	"testing"
	"time"

	"unwind/internal/plan"
)

func TestDailyBudgetStopsModelCallsAtLimit(t *testing.T) {
	calls := 0
	primary := generatorFunc(func(context.Context, int, string) (plan.Reply, error) {
		calls++
		return plan.Reply{Message: "今天辛苦了", Action: "喝杯水", Source: "ai"}, nil
	})
	budget := NewDailyBudget(primary, 1)
	if _, err := budget.Generate(context.Background(), 1, "今天很累"); err != nil {
		t.Fatal(err)
	}
	if _, err := budget.Generate(context.Background(), 1, "还想再试一次"); !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("second call error = %v, want ErrBudgetExceeded", err)
	}
	if calls != 1 {
		t.Fatalf("model calls = %d, want 1", calls)
	}
}

func TestDailyBudgetResetsOnNextDay(t *testing.T) {
	primary := generatorFunc(func(context.Context, int, string) (plan.Reply, error) {
		return plan.Reply{Message: "今天辛苦了", Action: "喝杯水", Source: "ai"}, nil
	})
	budget := NewDailyBudget(primary, 1).(*DailyBudget)
	now := time.Date(2026, 11, 20, 23, 0, 0, 0, time.Local)
	budget.now = func() time.Time { return now }
	if _, err := budget.Generate(context.Background(), 1, "今天"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Hour)
	if _, err := budget.Generate(context.Background(), 1, "明天"); err != nil {
		t.Fatalf("next day should reset budget: %v", err)
	}
}

func TestHighRiskInputDoesNotConsumeBudget(t *testing.T) {
	calls := 0
	primary := generatorFunc(func(context.Context, int, string) (plan.Reply, error) {
		calls++
		return plan.Reply{Message: "今天辛苦了", Action: "喝杯水", Source: "ai"}, nil
	})
	generator := NewSafe(NewDailyBudget(primary, 1))
	if _, err := generator.Generate(context.Background(), 1, "我不想活了"); err != nil {
		t.Fatal(err)
	}
	response, err := generator.Generate(context.Background(), 1, "普通烦恼")
	if err != nil || response.Source != "ai" || calls != 1 {
		t.Fatalf("safety input consumed budget: response=%+v calls=%d err=%v", response, calls, err)
	}
}
