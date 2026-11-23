package main

import "testing"

func TestEnvInt(t *testing.T) {
	t.Setenv("UNWIND_TEST_LIMIT", "")
	if got, err := envInt("UNWIND_TEST_LIMIT", 3); err != nil || got != 3 {
		t.Fatalf("default = %d, err=%v", got, err)
	}
	t.Setenv("UNWIND_TEST_LIMIT", "20")
	if got, err := envInt("UNWIND_TEST_LIMIT", 3); err != nil || got != 20 {
		t.Fatalf("configured = %d, err=%v", got, err)
	}
	t.Setenv("UNWIND_TEST_LIMIT", "invalid")
	if _, err := envInt("UNWIND_TEST_LIMIT", 3); err == nil {
		t.Fatal("invalid integer should fail")
	}
}
