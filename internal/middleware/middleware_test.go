package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	myjwt "wishlist-api/internal/jwt"
)

func TestAuthSchemeIsCaseInsensitiveAndStrict(t *testing.T) {
	manager := myjwt.NewManager("0123456789abcdef0123456789abcdef", time.Hour)
	token, err := manager.Generate(42)
	if err != nil {
		t.Fatal(err)
	}

	handler := Auth(manager)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := UserID(r.Context()); got != 42 {
			t.Fatalf("user id = %d, want 42", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	for _, scheme := range []string{"Bearer", "bearer", "BEARER"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", scheme+" "+token)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Errorf("scheme %q: status=%d body=%s", scheme, w.Code, w.Body.String())
		}
	}

	for _, header := range []string{"", "Basic " + token, "Bearer", "Bearer " + token + " extra"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", header)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("header %q: status=%d", header, w.Code)
		}
	}
}

func TestTimeoutReturnsJSONAndDiscardsLateResponse(t *testing.T) {
	release := make(chan struct{})
	finished := make(chan struct{})
	handler := Timeout(10 * time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		defer close(finished)
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusGatewayTimeout || w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("status=%d content-type=%q body=%s", w.Code, w.Header().Get("Content-Type"), w.Body.String())
	}
	body := w.Body.String()
	close(release)
	<-finished
	if w.Code != http.StatusGatewayTimeout || w.Body.String() != body {
		t.Fatalf("late response changed output: status=%d body=%s", w.Code, w.Body.String())
	}
}
