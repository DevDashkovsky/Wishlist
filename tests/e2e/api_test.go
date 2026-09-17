package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type client struct {
	base string
	http *http.Client
}

func (c client) request(method, path, token, body string) (int, []byte, error) {
	req, err := http.NewRequest(method, c.base+path, strings.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	return resp.StatusCode, data, err
}

func (c client) check(t *testing.T, method, path, token, body string, want int) []byte {
	t.Helper()
	status, data, err := c.request(method, path, token, body)
	if err != nil {
		t.Fatal(err)
	}
	if status != want {
		t.Fatalf("%s %s: status %d, want %d: %s", method, path, status, want, data)
	}
	return data
}

func field(t *testing.T, data []byte, name string) string {
	t.Helper()
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatal(err)
	}
	var value string
	if err := json.Unmarshal(obj[name], &value); err != nil || value == "" {
		t.Fatalf("missing string %s in %s", name, data)
	}
	return value
}

func TestWishlistLifecycle(t *testing.T) {
	base := os.Getenv("WISHLIST_E2E_URL")
	if base == "" {
		t.Skip("set WISHLIST_E2E_URL to run against an isolated HTTP server and database")
	}
	c := client{strings.TrimRight(base, "/"), &http.Client{Timeout: 10 * time.Second}}
	c.check(t, "GET", "/health", "", "", 200)
	c.check(t, "GET", "/ready", "", "", 200)
	const api = "/api/v1"
	email := "e2e-" + uuid.NewString() + "@example.com"
	credentials := fmt.Sprintf(`{"email":%q,"password":"test-password-123"}`, email)
	token := field(t, c.check(t, "POST", api+"/auth/register", "", credentials, 201), "token")
	c.check(t, "POST", api+"/auth/register", "", credentials, 409)
	c.check(t, "POST", api+"/auth/login", "", credentials, 200)
	c.check(t, "POST", api+"/auth/login", "", fmt.Sprintf(`{"email":%q,"password":"incorrect"}`, email), 401)
	other := field(t, c.check(t, "POST", api+"/auth/register", "", fmt.Sprintf(`{"email":%q,"password":"test-password-123"}`, "other-"+email), 201), "token")
	c.check(t, "GET", api+"/wishlists", "", "", 401)
	if got := c.check(t, "GET", api+"/wishlists", token, "", 200); strings.TrimSpace(string(got)) != "[]" {
		t.Fatalf("new user list = %s", got)
	}
	wishlist := c.check(t, "POST", api+"/wishlists", token, `{"title":"Birthday","event_date":"2027-06-15"}`, 201)
	id, share := field(t, wishlist, "id"), field(t, wishlist, "share_token")
	if got := field(t, wishlist, "event_date"); got != "2027-06-15" {
		t.Fatalf("event_date = %q", got)
	}
	path := api + "/wishlists/" + id
	public := api + "/shared/" + share
	t.Cleanup(func() { c.check(t, "DELETE", path, token, "", 204) })
	for _, request := range []struct{ path, token string }{{path, token}, {public, ""}} {
		data := c.check(t, "GET", request.path, request.token, "", 200)
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(data, &obj); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(obj["items"], []byte("[]")) {
			t.Fatalf("empty items should be [], got %s", data)
		}
		if request.path == public && (obj["share_token"] != nil || obj["user_id"] != nil) {
			t.Fatalf("private wishlist fields leaked: %s", data)
		}
	}
	c.check(t, "GET", path, other, "", 403)
	c.check(t, "PATCH", path, other, `{"title":"stolen"}`, 403)
	c.check(t, "DELETE", path, other, "", 403)
	for _, body := range []string{`{"title":""}`, `{"title":"   "}`, `{"event_date":"2027-02-30"}`, `null`, `{"title":"\u0000"}`} {
		c.check(t, "PATCH", path, token, body, 422)
	}
	for _, body := range []string{`{"title":"x"} {}`, `{"title":"x"}garbage`} {
		c.check(t, "PATCH", path, token, body, 400)
	}
	item := c.check(t, "POST", path+"/items", token, `{"title":"Keyboard","url":"https://example.com/keyboard"}`, 201)
	itemID := field(t, item, "id")
	itemPath := path + "/items/" + itemID
	c.check(t, "PATCH", itemPath, token, `{"priority":0}`, 422)
	c.check(t, "PUT", itemPath, token, `{"title":"Updated keyboard","priority":5}`, 200)

	for i := 0; i < 8; i++ {
		for _, p := range []string{path, itemPath} {
			a, b := fmt.Sprintf("title-%d", i), fmt.Sprintf("description-%d", i)
			var wg sync.WaitGroup
			for _, body := range []string{fmt.Sprintf(`{"title":%q}`, a), fmt.Sprintf(`{"description":%q}`, b)} {
				wg.Add(1)
				go func() {
					defer wg.Done()
					status, data, err := c.request("PATCH", p, token, body)
					if err != nil || status != 200 {
						t.Errorf("PATCH: status=%d body=%s err=%v", status, data, err)
					}
				}()
			}
			wg.Wait()
			data := c.check(t, "GET", path, token, "", 200)
			if p == itemPath {
				var obj struct {
					Items []json.RawMessage `json:"items"`
				}
				if err := json.Unmarshal(data, &obj); err != nil || len(obj.Items) != 1 {
					t.Fatalf("invalid items: %s", data)
				}
				data = obj.Items[0]
			}
			if field(t, data, "title") != a || field(t, data, "description") != b {
				t.Fatalf("lost PATCH update: %s", data)
			}
		}
	}

	second := c.check(t, "POST", api+"/wishlists", token, `{"title":"Other list","event_date":"2027-06-16"}`, 201)
	secondPath := api + "/wishlists/" + field(t, second, "id")
	t.Cleanup(func() { c.check(t, "DELETE", secondPath, token, "", 204) })
	c.check(t, "POST", api+"/shared/"+field(t, second, "share_token")+"/items/"+itemID+"/reserve", "", "", 404)
	c.check(t, "PATCH", secondPath+"/items/"+itemID, token, `{"title":"wrong list"}`, 404)

	statuses := make(chan int, 12)
	var wg sync.WaitGroup
	for i := 0; i < cap(statuses); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, data, err := c.request("POST", public+"/items/"+itemID+"/reserve", "", "")
			if err != nil {
				t.Errorf("reserve: %v (%s)", err, data)
			}
			statuses <- status
		}()
	}
	wg.Wait()
	close(statuses)
	winners := 0
	for status := range statuses {
		if status == 200 {
			winners++
		} else if status != 409 {
			t.Errorf("unexpected reserve status %d", status)
		}
	}
	if winners != 1 {
		t.Fatalf("reservation winners = %d, want 1", winners)
	}
	updated := c.check(t, "PATCH", itemPath, token, `{"description":"still reserved"}`, 200)
	var result struct {
		IsReserved bool `json:"is_reserved"`
	}
	if err := json.Unmarshal(updated, &result); err != nil || !result.IsReserved {
		t.Fatalf("reservation was lost: %s", updated)
	}
	c.check(t, "DELETE", itemPath, token, "", 204)
	c.check(t, "POST", public+"/items/"+itemID+"/reserve", "", "", 404)
}
