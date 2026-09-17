package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"wishlist-api/internal/domain"
)

const maxRequestBody = 1 << 20

var (
	errBodyTooLarge  = errors.New("request body too large")
	errMalformedJSON = errors.New("malformed JSON")
	errInvalidJSON   = errors.New("invalid JSON object")
)

func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic serving %s %s: %v\n%s", r.Method, r.URL.Path, recovered, debug.Stack())
				writeError(w, http.StatusInternalServerError, "something went wrong")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func LimitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(responseValue(v))
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func handleUnexpectedError(w http.ResponseWriter, err error) {
	if errors.Is(err, context.DeadlineExceeded) {
		writeError(w, http.StatusGatewayTimeout, "request timed out")
		return
	}
	log.Printf("internal error: %v", err)
	writeError(w, http.StatusInternalServerError, "something went wrong")
}

func decodeJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return fmt.Errorf("%w: missing body", errMalformedJSON)
	}
	defer func() { _ = r.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBody+1))
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return errBodyTooLarge
		}
		return fmt.Errorf("%w: %v", errMalformedJSON, err)
	}
	if len(data) > maxRequestBody {
		return errBodyTooLarge
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		var syntaxErr *json.SyntaxError
		if errors.As(err, &syntaxErr) {
			return fmt.Errorf("%w: %v", errMalformedJSON, err)
		}
		return fmt.Errorf("%w: %v", errInvalidJSON, err)
	}
	if fields == nil {
		return fmt.Errorf("%w: expected object", errInvalidJSON)
	}
	for _, raw := range fields {
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return fmt.Errorf("%w: null is not allowed", errInvalidJSON)
		}
		var str string
		if json.Unmarshal(raw, &str) == nil && strings.ContainsRune(str, 0) {
			return fmt.Errorf("%w: NUL is not allowed", errInvalidJSON)
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("%w: %v", errInvalidJSON, err)
	}
	return nil
}

func handleDecodeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errBodyTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
	case errors.Is(err, errMalformedJSON):
		writeError(w, http.StatusBadRequest, "malformed JSON")
	default:
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
	}
}

func wishlistResponse(w domain.Wishlist) any {
	type plain domain.Wishlist
	return struct {
		plain
		EventDate string `json:"event_date"`
	}{plain(w), w.EventDate.Format("2006-01-02")}
}

func responseValue(v any) any {
	switch value := v.(type) {
	case *domain.Wishlist:
		return wishlistResponse(*value)
	case []domain.Wishlist:
		result := make([]any, 0, len(value))
		for _, w := range value {
			result = append(result, wishlistResponse(w))
		}
		return result
	case *domain.WishlistWithItems:
		type plain domain.Wishlist
		return struct {
			plain
			EventDate string        `json:"event_date"`
			Items     []domain.Item `json:"items"`
		}{plain(value.Wishlist), value.EventDate.Format("2006-01-02"), value.Items}
	case *domain.PublicWishlistWithItems:
		type plain domain.PublicWishlist
		return struct {
			plain
			EventDate string        `json:"event_date"`
			Items     []domain.Item `json:"items"`
		}{plain(value.PublicWishlist), value.EventDate.Format("2006-01-02"), value.Items}
	default:
		return v
	}
}
