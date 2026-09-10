package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"wishlist-api/internal/domain"
)

const maxRequestBody = 1 << 20

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

func decodeJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return errors.New("missing body")
	}
	defer func() { _ = r.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBody+1))
	if err != nil {
		return err
	}
	if len(data) > maxRequestBody {
		return errors.New("body too large")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if fields == nil {
		return errors.New("expected object")
	}
	for _, raw := range fields {
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return errors.New("null is not allowed")
		}
		var str string
		if json.Unmarshal(raw, &str) == nil && strings.ContainsRune(str, 0) {
			return errors.New("NUL is not allowed")
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
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
