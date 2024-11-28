package handlers

import (
	"fmt"
	"net/http"
	"shop/handlers/render"
	"shop/services/auth"
	"shop/services/auth/authGoogle"
	"shop/services/store"
	"shop/views/login"
)

func AuthLogout(w http.ResponseWriter, r *http.Request) error {
	user, err := auth.GetUserSession(r)
	if err != nil {
		return err
	}
	var service auth.AuthService
	switch user.Provider {
	case store.Google:
		service = authGoogle.NewService()
	case store.Local:
	}
	if service == nil {
		err = auth.RemoveUserSession(w, r)
		if err != nil {
			return fmt.Errorf("service is nil, cant logout without a provider, has provider: %s, error: %w", user.Provider, err)
		}
		return fmt.Errorf("service is nil, cant logout without a provider, has provider: %s", user.Provider)
	}

	err = auth.LogOutUser(w, r, service, r.Context())
	if err != nil {
		return err
	}
	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusSeeOther)
	return nil
}

func LoginPage(w http.ResponseWriter, r *http.Request) error {
	return render.Template(w, r, login.Index())
}
