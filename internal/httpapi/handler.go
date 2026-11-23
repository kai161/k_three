package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"unwind/internal/plan"
	"unwind/internal/store"
)

func contextWithDevice(request *http.Request, deviceID string) context.Context {
	return context.WithValue(request.Context(), deviceContextKey{}, deviceID)
}

func deviceID(request *http.Request) string {
	value, _ := request.Context().Value(deviceContextKey{}).(string)
	return value
}

func (r *Router) handleSession(w http.ResponseWriter, request *http.Request) {
	status, err := r.service.Current(request.Context(), deviceID(request))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (r *Router) handlePlan(w http.ResponseWriter, request *http.Request) {
	r.handleSession(w, request)
}

func (r *Router) handleReply(w http.ResponseWriter, request *http.Request) {
	if r.dailyLimiter != nil && !r.dailyLimiter.Allow(deviceID(request)) {
		writeError(w, errRateLimited)
		return
	}
	if !r.limiter.Allow(deviceID(request)) {
		writeError(w, errRateLimited)
		return
	}
	var input struct {
		Text string `json:"text"`
	}
	if err := decodeJSON(w, request, &input); err != nil {
		writeError(w, err)
		return
	}
	reply, err := r.service.Generate(request.Context(), deviceID(request), input.Text)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reply)
}

func (r *Router) handleEntry(w http.ResponseWriter, request *http.Request) {
	var input struct {
		Day    int    `json:"day"`
		Mood   string `json:"mood"`
		Ritual string `json:"ritual"`
	}
	if err := decodeJSON(w, request, &input); err != nil {
		writeError(w, err)
		return
	}
	if !contains([]string{"轻松", "平静", "复杂", "疲惫"}, input.Mood) || !contains([]string{"揉掉", "封存"}, input.Ritual) {
		writeError(w, plan.ErrInvalidInput)
		return
	}
	status, err := r.service.Complete(request.Context(), deviceID(request), input.Day, input.Mood, input.Ritual)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (r *Router) handleActivation(w http.ResponseWriter, request *http.Request) {
	var input struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(w, request, &input); err != nil {
		writeError(w, err)
		return
	}
	status, err := r.service.Activate(request.Context(), deviceID(request), input.Code)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (r *Router) handleFeedback(w http.ResponseWriter, request *http.Request) {
	var input struct {
		Day  int    `json:"day"`
		Text string `json:"text"`
	}
	if err := decodeJSON(w, request, &input); err != nil {
		writeError(w, err)
		return
	}
	if err := r.service.SaveFeedback(request.Context(), deviceID(request), input.Day, input.Text); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (r *Router) handleReview(w http.ResponseWriter, request *http.Request) {
	review, err := r.service.Review(request.Context(), deviceID(request))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, review)
}

func decodeJSON(w http.ResponseWriter, request *http.Request, destination any) error {
	request.Body = http.MaxBytesReader(w, request.Body, 16<<10)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return plan.ErrInvalidInput
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return plan.ErrInvalidInput
	}
	return nil
}

func writeError(w http.ResponseWriter, err error) {
	status := errorStatus(err)
	message := map[int]string{
		http.StatusBadRequest:          "请检查输入内容后再试。",
		http.StatusPaymentRequired:     "两天体验已完成，激活后即可继续。",
		http.StatusConflict:            "请按顺序完成今天的仪式。",
		http.StatusTooManyRequests:     "今天可以稍慢一点，请一分钟后再试。",
		http.StatusInternalServerError: "暂时没有回应成功，请稍后重试。",
	}[status]
	if errors.Is(err, store.ErrInvalidActivation) {
		message = "激活码无效或已被使用，请核对后再试。"
	}
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
