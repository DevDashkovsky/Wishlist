package handler

import (
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"wishlist-api/internal/domain"
	"wishlist-api/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input domain.RegisterInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	input.Email = strings.TrimSpace(strings.ToLower(input.Email))

	if !validEmail(input.Email) {
		writeError(w, http.StatusUnprocessableEntity, "invalid email format")
		return
	}
	if len(input.Password) < 8 || len(input.Password) > 72 {
		writeError(w, http.StatusUnprocessableEntity, "password must contain 8 to 72 bytes")
		return
	}

	token, err := h.auth.Register(r.Context(), input)
	if errors.Is(err, service.ErrEmailTaken) {
		writeError(w, http.StatusConflict, "email already taken")
		return
	}
	if err != nil {
		log.Printf("internal error: %v", err)
		writeError(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"token": token})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input domain.LoginInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	input.Email = strings.TrimSpace(strings.ToLower(input.Email))

	if !validEmail(input.Email) || len(input.Password) < 8 || len(input.Password) > 72 {
		writeError(w, http.StatusUnprocessableEntity, "invalid email or password format")
		return
	}

	token, err := h.auth.Login(r.Context(), input)
	if errors.Is(err, service.ErrInvalidCredentials) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err != nil {
		log.Printf("internal error: %v", err)
		writeError(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func validEmail(email string) bool {
	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email && strings.Contains(email, "@")
}
