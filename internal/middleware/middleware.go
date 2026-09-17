package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	myjwt "wishlist-api/internal/jwt"
)

type contextKey string

const userIDKey contextKey = "user_id"

func Timeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			buffered := newTimeoutWriter()
			done := make(chan struct{})
			go func() {
				defer close(done)
				next.ServeHTTP(buffered, r.WithContext(ctx))
			}()

			select {
			case <-done:
				if ctx.Err() != nil {
					buffered.timeout(w)
					return
				}
				buffered.flush(w)
			case <-ctx.Done():
				buffered.timeout(w)
			}
		})
	}
}

type timeoutWriter struct {
	mu       sync.Mutex
	header   http.Header
	body     bytes.Buffer
	status   int
	timedOut bool
}

func newTimeoutWriter() *timeoutWriter {
	return &timeoutWriter{header: make(http.Header)}
}

func (w *timeoutWriter) Header() http.Header {
	return w.header
}

func (w *timeoutWriter) WriteHeader(status int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timedOut || w.status != 0 {
		return
	}
	w.status = status
}

func (w *timeoutWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timedOut {
		return 0, http.ErrHandlerTimeout
	}
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(data)
}

func (w *timeoutWriter) flush(dst http.ResponseWriter) {
	w.mu.Lock()
	defer w.mu.Unlock()
	copyHeader(dst.Header(), w.header)
	status := w.status
	if status == 0 {
		status = http.StatusOK
	}
	dst.WriteHeader(status)
	_, _ = dst.Write(w.body.Bytes())
}

func (w *timeoutWriter) timeout(dst http.ResponseWriter) {
	w.mu.Lock()
	w.timedOut = true
	w.mu.Unlock()
	dst.Header().Set("Content-Type", "application/json")
	dst.WriteHeader(http.StatusGatewayTimeout)
	_, _ = dst.Write([]byte("{\"error\":\"request timed out\"}\n"))
}

func copyHeader(dst, src http.Header) {
	for key, values := range src {
		dst[key] = append([]string(nil), values...)
	}
}

func Auth(jwtManager *myjwt.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				unauthorized(w)
				return
			}

			parts := strings.Fields(header)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				unauthorized(w)
				return
			}

			userID, err := jwtManager.Parse(parts[1])
			if err != nil {
				unauthorized(w)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserID(ctx context.Context) int64 {
	id, _ := ctx.Value(userIDKey).(int64)
	return id
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}
