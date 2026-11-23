package reply

import "strings"

var highRiskPhrases = []string{
	"不想活",
	"自杀",
	"自残",
	"伤害自己",
	"结束生命",
	"kill myself",
	"hurt myself",
	"end my life",
}

func IsHighRisk(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	for _, phrase := range highRiskPhrases {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	return false
}
