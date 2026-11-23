package reply

import (
	"context"
	"errors"
	"testing"

	"unwind/internal/plan"
)

type generatorFunc func(context.Context, int, string) (plan.Reply, error)

func (f generatorFunc) Generate(ctx context.Context, day int, text string) (plan.Reply, error) {
	return f(ctx, day, text)
}

func TestSafeGeneratorFallsBackWithoutClaimingAI(t *testing.T) {
	primary := generatorFunc(func(context.Context, int, string) (plan.Reply, error) {
		return plan.Reply{}, errors.New("timeout")
	})
	generator := NewSafe(primary)
	response, err := generator.Generate(context.Background(), 1, "今天很累")
	if err != nil {
		t.Fatal(err)
	}
	if response.Source != "human" || response.Message == "" || response.Action == "" {
		t.Fatalf("unexpected fallback: %+v", response)
	}
}

func TestSafeGeneratorBypassesModelForHighRiskInput(t *testing.T) {
	called := false
	primary := generatorFunc(func(context.Context, int, string) (plan.Reply, error) {
		called = true
		return plan.Reply{}, nil
	})
	response, err := NewSafe(primary).Generate(context.Background(), 1, "我不想活了")
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("high-risk input must not be sent to the model")
	}
	if response.Source != "safety" {
		t.Fatalf("source = %q, want safety", response.Source)
	}
}
