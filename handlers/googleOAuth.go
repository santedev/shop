package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"shop/services/auth"
	"shop/services/auth/authGoogle"
	"shop/services/store"

	"google.golang.org/api/idtoken"
)

func HandleCredentialsGoogle(w http.ResponseWriter, r *http.Request) error {
	var reader map[string]any
	if err := json.NewDecoder(r.Body).Decode(&reader); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return err
	}

	cred, ok := reader["credential"].(string)
	if !ok {
		http.Error(w, "Missing or invalid credential", http.StatusBadRequest)
		return fmt.Errorf("missing or invalid credential")
	}

	payload, err := authGoogle.VerifyIdToken(cred)
	if err != nil {
		http.Error(w, "Failed to verify ID token", http.StatusInternalServerError)
		return err
	}

	user, err := formatToUser(payload)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to format user: %s", err.Error()), http.StatusBadRequest)
		return err
	}

	ctx := context.WithValue(r.Context(), "user", user)
	_, err = auth.StoreUser(w, r, authGoogle.NewService(), ctx)
	if err != nil {
		http.Error(w, "failed to store user", http.StatusInternalServerError)
		return err
	}

	redirectResponse("", "/", http.StatusOK, w)
	return nil
}

func redirectResponse(body, redirectURL string, code int, w http.ResponseWriter) {
	response := map[string]string{"redirect_url": redirectURL, "context": body}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(response)
}

func formatToUser(payload *idtoken.Payload) (store.User, error) {
	name, ok := payload.Claims["name"].(string)
	if !ok {
		return store.User{}, errors.New("not name found within payload claims")
	}
	email, ok := payload.Claims["email"].(string)
	if !ok {
		return store.User{}, errors.New("not email found within payload claims")
	}
	picture, ok := payload.Claims["picture"].(string)
	if !ok {
		return store.User{}, errors.New("not picture found within payload claims")
	}
	return store.User{Name: name, Email: email, AvatarUrl: picture, Provider: store.Google}, nil
}
