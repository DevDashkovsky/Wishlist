package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"wishlist-api/internal/domain"
)

func TestDecodeJSON(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `{}`, `{"title":"ok"} {}`, `{"title":"ok"} trailing`, `{"unknown":1}`, `{"title":null}`, `{"title":"a\u0000b"}`, `{"title":"ok"}` + strings.Repeat(" ", maxRequestBody)} {
		t.Run(body[:min(40, len(body))], func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
			var input domain.ItemInput
			err := decodeJSON(r, &input)
			if body == `{}` {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("invalid JSON accepted")
			}
		})
	}
	input := domain.ItemInput{Priority: 3}
	if err := decodeJSON(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"title":"ok","priority":0}`)), &input); err != nil {
		t.Fatal(err)
	}
	if input.Priority != 0 {
		t.Fatal("explicit zero replaced by default")
	}
}

func TestDecodeErrorsUseDistinctStatuses(t *testing.T) {
	auth := NewAuthHandler(nil)
	router := chi.NewRouter()
	router.Use(LimitBody)
	router.Post("/register", auth.Register)

	for _, tc := range []struct {
		name string
		body string
		want int
	}{
		{name: "malformed", body: `{"email":`, want: http.StatusBadRequest},
		{name: "semantic", body: `{"unknown":true}`, want: http.StatusUnprocessableEntity},
		{name: "too large", body: `{"email":"` + strings.Repeat("a", maxRequestBody) + `"}`, want: http.StatusRequestEntityTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(tc.body)))
			if w.Code != tc.want {
				t.Fatalf("got %d, want %d: %s", w.Code, tc.want, w.Body.String())
			}
		})
	}
}

func TestRecovererReturnsJSONError(t *testing.T) {
	h := Recoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if w.Code != http.StatusInternalServerError || w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("unexpected response: status=%d content-type=%q body=%s", w.Code, w.Header().Get("Content-Type"), w.Body.String())
	}
}

func TestReadinessReflectsDatabaseState(t *testing.T) {
	dbDown := errors.New("database down")
	for _, tc := range []struct {
		name      string
		readiness func(context.Context) error
		want      int
	}{
		{name: "ready", readiness: func(context.Context) error { return nil }, want: http.StatusOK},
		{name: "not ready", readiness: func(context.Context) error { return dbDown }, want: http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := NewRouter(time.Second, tc.readiness, nil, nil, nil, nil, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ready", nil))
			if w.Code != tc.want {
				t.Fatalf("got %d, want %d: %s", w.Code, tc.want, w.Body.String())
			}
		})
	}
}

func TestInvalidInputsReturn422(t *testing.T) {
	auth := NewAuthHandler(nil)
	wishlist := NewWishlistHandler(nil)
	item := NewItemHandler(nil)
	router := chi.NewRouter()
	router.Post("/register", auth.Register)
	router.Post("/login", auth.Login)
	router.Post("/wishlists", wishlist.Create)
	router.Put("/wishlists/{id}", wishlist.Update)
	router.Patch("/wishlists/{id}", wishlist.Patch)
	router.Post("/wishlists/{id}/items", item.Create)
	router.Put("/wishlists/{id}/items/{itemId}", item.Update)
	router.Patch("/wishlists/{id}/items/{itemId}", item.Patch)
	id := uuid.NewString()
	cases := []struct{ method, path, body string }{
		{"POST", "/register", `{"email":"@","password":"password"}`},
		{"POST", "/register", `{"email":"ok@example.com","password":"` + strings.Repeat("a", 73) + `"}`},
		{"POST", "/login", `{"email":"ok@example.com","password":"` + strings.Repeat("a", 73) + `"}`},
		{"POST", "/wishlists", `{"title":"  ","event_date":"2026-01-01"}`},
		{"PUT", "/wishlists/" + id, `{"title":"  ","event_date":"2026-01-01"}`},
		{"PATCH", "/wishlists/" + id, `{"title":"  "}`},
		{"POST", "/wishlists/" + id + "/items", `{"title":"ok","priority":0}`},
		{"PUT", "/wishlists/" + id + "/items/" + id, `{"title":"ok","priority":0}`},
		{"PATCH", "/wishlists/" + id + "/items/" + id, `{"title":"  "}`},
	}
	for _, method := range []string{"POST", "PUT", "PATCH"} {
		path := "/wishlists/" + id + "/items"
		if method != "POST" {
			path += "/" + id
		}
		for _, value := range []string{"not a URL", "/relative", "https://", "http:///path"} {
			body, err := json.Marshal(map[string]any{"title": "ok", "url": value})
			if err != nil {
				t.Fatal(err)
			}
			cases = append(cases, struct{ method, path, body string }{method, path, string(body)})
		}
	}

	for _, tc := range cases {
		t.Run(tc.method+tc.path+tc.body[:10], func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body)))
			if w.Code != http.StatusUnprocessableEntity {
				t.Fatalf("got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestWishlistResponsesPreserveItemsAndDate(t *testing.T) {
	date := time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC)
	for _, value := range []any{
		&domain.Wishlist{EventDate: date},
		&domain.WishlistWithItems{Wishlist: domain.Wishlist{EventDate: date}, Items: []domain.Item{}},
		&domain.PublicWishlistWithItems{PublicWishlist: domain.PublicWishlist{EventDate: date}, Items: []domain.Item{}},
	} {
		w := httptest.NewRecorder()
		writeJSON(w, 200, value)
		var decoded map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded["event_date"] != "2026-09-09" {
			t.Fatal(w.Body.String())
		}
		if _, simple := value.(*domain.Wishlist); !simple {
			items, ok := decoded["items"].([]any)
			if !ok || len(items) != 0 {
				t.Fatal(w.Body.String())
			}
		}
	}
}

func TestValidURL(t *testing.T) {
	for _, value := range []string{"", "https://example.com/gift?q=1", "http://localhost:8080/gift"} {
		if !validURL(value) {
			t.Errorf("valid URI rejected: %q", value)
		}
	}
	for _, value := range []string{"not a URL", "/relative", "https://", "http:///path", "https://exa mple.com", "https://example.com/%ZZ", "javascript:alert(1)", "data:text/html,x", "mailto:test@example.com", "https://user:pass@example.com/gift"} {
		if validURL(value) {
			t.Errorf("invalid URI accepted: %q", value)
		}
	}
}
