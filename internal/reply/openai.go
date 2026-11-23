package reply

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"unwind/internal/plan"
)

// OpenAI calls an OpenAI-compatible chat completions endpoint.
type OpenAI struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func NewOpenAI(baseURL, apiKey, model string, client *http.Client) *OpenAI {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &OpenAI{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, model: model, client: client}
}

func (g *OpenAI) Generate(ctx context.Context, day int, text string) (plan.Reply, error) {
	payload := map[string]any{
		"model": g.model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": fmt.Sprintf("第%d天。用户的记录：%s", day, text)},
		},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":     0.7,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return plan.Reply{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return plan.Reply{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+g.apiKey)
	response, err := g.client.Do(request)
	if err != nil {
		return plan.Reply{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return plan.Reply{}, fmt.Errorf("model returned status %d", response.StatusCode)
	}
	var envelope struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&envelope); err != nil {
		return plan.Reply{}, err
	}
	if len(envelope.Choices) == 0 {
		return plan.Reply{}, fmt.Errorf("model returned no choices")
	}
	var reply plan.Reply
	if err := json.Unmarshal([]byte(envelope.Choices[0].Message.Content), &reply); err != nil {
		return plan.Reply{}, fmt.Errorf("invalid structured reply: %w", err)
	}
	reply.Source = "ai"
	if !valid(reply) {
		return plan.Reply{}, fmt.Errorf("model reply failed validation")
	}
	return reply, nil
}

const systemPrompt = `你是一位克制、温暖的晚间情绪陪伴者。不要说教，不要夸大痛苦，不要诊断或承诺治疗效果。用中文输出 JSON：message 是 40～100 字的回应，action 是一个 30 字以内、可立即完成的小行动。若用户表达自伤风险，不要按普通情绪回应。`
