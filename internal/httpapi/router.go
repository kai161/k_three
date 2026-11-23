package httpapi

import (
	"errors"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"unwind/internal/plan"
	"unwind/internal/store"
)

// Router connects anonymous sessions, the plan service and static assets.
type Router struct {
	service      *plan.Service
	session      *SessionManager
	limiter      *replyLimiter
	dailyLimiter *replyLimiter
	api          *http.ServeMux
	web          http.Handler
}

// Config controls public-beta behavior while preserving the original defaults.
type Config struct {
	PublicBeta      bool
	DailyReplyLimit int
}

func NewRouter(store plan.Store, generator plan.Generator, web fs.FS, secure bool) http.Handler {
	return NewRouterWithConfig(store, generator, web, secure, Config{})
}

func NewRouterWithConfig(store plan.Store, generator plan.Generator, web fs.FS, secure bool, config Config) http.Handler {
	router := &Router{
		service: plan.NewServiceWithOptions(store, generator, plan.Options{PublicBeta: config.PublicBeta}),
		session: NewSessionManager(store, secure),
		limiter: newReplyLimiter(10, time.Minute),
		api:     http.NewServeMux(),
		web:     http.FileServer(http.FS(web)),
	}
	if config.PublicBeta {
		if config.DailyReplyLimit <= 0 {
			config.DailyReplyLimit = 3
		}
		router.dailyLimiter = newReplyLimiter(config.DailyReplyLimit, 24*time.Hour)
	}
	router.routes()
	return router
}

func (r *Router) routes() {
	r.api.HandleFunc("POST /api/session", r.handleSession)
	r.api.HandleFunc("GET /api/plan", r.handlePlan)
	r.api.HandleFunc("POST /api/replies", r.handleReply)
	r.api.HandleFunc("POST /api/entries", r.handleEntry)
	r.api.HandleFunc("POST /api/activate", r.handleActivation)
	r.api.HandleFunc("POST /api/feedback", r.handleFeedback)
	r.api.HandleFunc("GET /api/review", r.handleReview)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	if request.URL.Path == "/healthz" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	if request.URL.Path == "/api/" || len(request.URL.Path) > 5 && request.URL.Path[:5] == "/api/" {
		r.withSession(w, request, func(deviceID string) {
			request = withDevice(request, deviceID)
			r.api.ServeHTTP(w, request)
		})
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	r.web.ServeHTTP(w, request)
}

func (r *Router) withSession(w http.ResponseWriter, request *http.Request, next func(string)) {
	deviceID, err := r.session.Ensure(w, request)
	if err != nil {
		writeError(w, err)
		return
	}
	next(deviceID)
}

type deviceContextKey struct{}

func withDevice(request *http.Request, deviceID string) *http.Request {
	return request.WithContext(contextWithDevice(request, deviceID))
}

func errorStatus(err error) int {
	switch {
	case errors.Is(err, plan.ErrInvalidInput), errors.Is(err, store.ErrInvalidActivation):
		return http.StatusBadRequest
	case errors.Is(err, plan.ErrLocked):
		return http.StatusPaymentRequired
	case errors.Is(err, plan.ErrInvalidDay), errors.Is(err, plan.ErrNotFinished):
		return http.StatusConflict
	case errors.Is(err, errRateLimited):
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

var errRateLimited = errors.New("too many replies")

type replyWindow struct {
	started time.Time
	count   int
}

type replyLimiter struct {
	mu      sync.Mutex
	windows map[string]*replyWindow
	limit   int
	period  time.Duration
	now     func() time.Time
}

func newReplyLimiter(limit int, period time.Duration) *replyLimiter {
	return &replyLimiter{windows: make(map[string]*replyWindow), limit: limit, period: period, now: time.Now}
}

func (l *replyLimiter) Allow(deviceID string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	window := l.windows[deviceID]
	if window == nil || now.Sub(window.started) >= l.period {
		l.windows[deviceID] = &replyWindow{started: now, count: 1}
		return true
	}
	if window.count >= l.limit {
		return false
	}
	window.count++
	return true
}
