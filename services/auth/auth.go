package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"shop/services/store"
	"strconv"

	"github.com/jackc/pgx/v5"
)

type AuthService interface {
	ValidUser(context.Context) error
	AuthUser(context.Context) (store.User, error)
	Logout(context.Context) error
	RemoveUser(context.Context) error
	DeleteUser(context.Context) error
}

func StoreUser(w http.ResponseWriter, r *http.Request, auth AuthService, ctx context.Context) (store.User, error) {
	user, err := auth.AuthUser(ctx)
	if err != nil {
		return store.User{}, err
	}
	_, err = store.Pub.GetUser(ctx)
	if err == nil {
		user, err := store.Pub.RestoreUser(ctx)
		if err != nil {
			return store.User{}, err
		}
		err = SetUserSession(w, r, user)
		if err != nil {
			return store.User{}, err
		}
		return user, nil
	}
	if err != pgx.ErrNoRows {
		return store.User{}, err
	}

	user, err = store.Pub.NewUser(ctx, user)
	if err != nil {
		return store.User{}, err
	}
	err = SetUserSession(w, r, user)
	if err != nil {
		return store.User{}, err
	}
	return user, nil
}

func GetUser(r *http.Request, auth AuthService, ctx context.Context) (store.User, error) {
	user, err := GetUserSession(r)
	if err != ErrNoUserSessionFound && err != nil {
		return store.User{}, err
	}
	if err := auth.ValidUser(ctx); err != nil {
		return store.User{}, err
	}
	if err == ErrNoUserSessionFound {
		user, err = store.Pub.RestoreUser(ctx)
		if err != nil {
			return store.User{}, err
		}
	}
	return user, nil
}

func DeleteUser(w http.ResponseWriter, r *http.Request, auth AuthService, ctx context.Context) error {
	user, ok := ctx.Value("user").(store.User)
	if !ok {
		return errors.New("context value user was not passed for deleting the user")
	}
	if user.Id <= 0 {
		return errors.New("user uid is less than zero, invalid for deletion")
	}
	err := RemoveUserSession(w, r)
	if err != nil {
		return err
	}
	err = auth.DeleteUser(ctx)
	if err != nil {
		return err
	}
	err = store.Pub.DeleteUser(ctx, strconv.Itoa(int(user.Id)))
	if err != nil {
		return fmt.Errorf("cant delete user with err: %w", err)
	}
	return nil
}

func LogOutUser(w http.ResponseWriter, r *http.Request, auth AuthService, ctx context.Context) error {
	err := RemoveUserSession(w, r)
	if err != nil {
		return err
	}
	err = auth.RemoveUser(ctx)
	if err != nil {
		return err
	}
	return nil
}
