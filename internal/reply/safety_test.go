package reply

import "testing"

func TestHighRiskDetection(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"今天会议很多，有点累", false},
		{"我不想活了", true},
		{"我想伤害自己", true},
		{"I want to kill myself", true},
	}
	for _, test := range tests {
		if got := IsHighRisk(test.text); got != test.want {
			t.Errorf("IsHighRisk(%q) = %v, want %v", test.text, got, test.want)
		}
	}
}
