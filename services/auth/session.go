package auth

import (
	"errors"
	"net/http"
	"shop/services/store"

	"github.com/gorilla/sessions"
)

var ErrNoUserSessionFound = errors.New("no user found in session")

const sessionName = "user_session"

var userCookie *sessions.CookieStore

type SessionOptions struct {
	CookiesKey string
	Path       string
	MaxAge     int
	HttpOnly   bool
	Secure     bool
	SameSite   http.SameSite
}

func NewUserCookieStore(opts SessionOptions) {
	userCookie = sessions.NewCookieStore([]byte(opts.CookiesKey))
	userCookie.Options.Path = opts.Path
	userCookie.Options.MaxAge = opts.MaxAge
	userCookie.Options.HttpOnly = opts.HttpOnly
	userCookie.Options.Secure = opts.Secure
	userCookie.Options.SameSite = opts.SameSite
}

func SetUserSession(w http.ResponseWriter, r *http.Request, account store.Account) error {
	session, err := userCookie.Get(r, sessionName)
	if err != nil {
		return err
	}
	switch account.(type) {
	case store.User:
		session.Values["user"] = account
	case store.Admin:
		session.Values["admin"] = account
	default:
		return errors.New("invalid struct for interface account")
	}

	err = session.Save(r, w)
	if err != nil {
		return err
	}

	return nil
}

func GetUserSession(r *http.Request) (store.Account, error) {
	session, err := userCookie.Get(r, sessionName)
	if err != nil {
		return store.User{}, err
	}

	user, ok := session.Values["user"].(store.User)
	if ok {
		return user, nil
	}
	admin, ok := session.Values["admin"].(store.Admin)
	if ok {
		return admin, nil
	}
	return user, ErrNoUserSessionFound
}

func RemoveUserSession(w http.ResponseWriter, r *http.Request) error {
	session, err := userCookie.Get(r, sessionName)
	if err != nil {
		return err
	}

	session.Options.MaxAge = -1
	return session.Save(r, w)
}
