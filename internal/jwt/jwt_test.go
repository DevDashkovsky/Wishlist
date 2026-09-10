package jwt

import (
	jwtlib "github.com/golang-jwt/jwt/v5"
	"testing"
	"time"
)

func TestParseRejectsInvalidClaimsAndAlgorithms(t *testing.T) {
	manager := NewManager("test-secret", time.Hour)
	for _, tc := range []struct {
		name   string
		method jwtlib.SigningMethod
		sub    any
		exp    any
	}{
		{"missing expiration", jwtlib.SigningMethodHS256, "1", nil},
		{"expired", jwtlib.SigningMethodHS256, "1", time.Now().Add(-time.Hour).Unix()},
		{"wrong algorithm", jwtlib.SigningMethodHS384, "1", time.Now().Add(time.Hour).Unix()},
		{"zero subject", jwtlib.SigningMethodHS256, "0", time.Now().Add(time.Hour).Unix()},
		{"negative subject", jwtlib.SigningMethodHS256, "-1", time.Now().Add(time.Hour).Unix()},
		{"numeric subject", jwtlib.SigningMethodHS256, 1, time.Now().Add(time.Hour).Unix()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claims := jwtlib.MapClaims{"sub": tc.sub}
			if tc.exp != nil {
				claims["exp"] = tc.exp
			}
			token, err := jwtlib.NewWithClaims(tc.method, claims).SignedString([]byte("test-secret"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := manager.Parse(token); err == nil {
				t.Fatal("invalid token accepted")
			}
		})
	}
	token, err := manager.Generate(42)
	if err != nil {
		t.Fatal(err)
	}
	if id, err := manager.Parse(token); err != nil || id != 42 {
		t.Fatalf("id=%d err=%v", id, err)
	}
}
