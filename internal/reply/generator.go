package reply

import (
	"context"
	"strings"
	"unicode/utf8"

	"unwind/internal/plan"
)

var curated = []plan.Reply{
	{Message: "今天已经结束了。你不必在此刻解决所有问题，先把这一晚还给自己。", Action: "把手机放远一点，慢慢喝几口水。", Source: "human"},
	{Message: "那些没处理完的事，可以等到明天。此刻的你，不需要继续证明自己。", Action: "让肩膀松下来，做三次缓慢呼吸。", Source: "human"},
	{Message: "累并不说明你做得不够好，它只是提醒你：今天消耗了很多。", Action: "调暗一盏灯，给自己两分钟安静。", Source: "human"},
	{Message: "你可以承认今天有点难，也可以暂时不去想它。两种选择都没有错。", Action: "写下明天最先做的一件小事，然后合上笔记。", Source: "human"},
	{Message: "工作占据过你的时间，但不必继续占据整个夜晚。", Action: "选一个喜欢的声音，安静地听一分钟。", Source: "human"},
	{Message: "不必急着把情绪整理好。能被看见，就已经是一种照顾。", Action: "轻轻伸展脖子和手腕，感受身体慢慢放松。", Source: "human"},
	{Message: "这一周，你一次次把注意力带回自己。今晚，也可以安心地休息。", Action: "留下一句想对明天的自己说的话。", Source: "human"},
}

// Safe prevents high-risk text from entering the model and provides curated fallback replies.
type Safe struct {
	primary plan.Generator
}

func NewSafe(primary plan.Generator) *Safe {
	return &Safe{primary: primary}
}

func (s *Safe) Generate(ctx context.Context, day int, text string) (plan.Reply, error) {
	if IsHighRisk(text) {
		return plan.Reply{
			Message: "看到你正在承受很强的痛苦。此刻请不要独自承担，立即联系一位你信任的人陪着你，并联系当地心理援助或医疗机构。如果你有立即伤害自己的危险，请马上联系当地急救或报警服务。",
			Action:  "先离开可能伤害自己的物品，去一个有人的、安全的地方。",
			Source:  "safety",
		}, nil
	}
	if s.primary != nil {
		response, err := s.primary.Generate(ctx, day, text)
		if err == nil && valid(response) {
			return response, nil
		}
	}
	index := day - 1
	if index < 0 || index >= len(curated) {
		index = 0
	}
	return curated[index], nil
}

func valid(response plan.Reply) bool {
	messageLength := utf8.RuneCountInString(strings.TrimSpace(response.Message))
	actionLength := utf8.RuneCountInString(strings.TrimSpace(response.Action))
	return messageLength > 0 && messageLength <= 280 && actionLength > 0 && actionLength <= 120
}
