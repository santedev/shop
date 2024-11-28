package authGoogle

import (
	"context"
	"errors"
	"shop/config"
	"shop/services/store"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
)

type googleService struct {
	service *oauth2.Config
}

func (g googleService) ValidUser(context.Context) error {
	return nil
}
func (g googleService) AuthUser(ctx context.Context) (store.User, error) {
	user, ok := ctx.Value("user").(store.User)
	if !ok {
		return store.User{}, errors.New("not a user, invalid user from context")
	}
	user.Provider = store.Google
	return user, nil
}
func (g googleService) Logout(context.Context) error {
	return nil
}
func (g googleService) RemoveUser(context.Context) error {
	return nil
}
func (g googleService) DeleteUser(context.Context) error {
	return nil
}

var scopes = []string{
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
}

func NewService() googleService {
	config := &oauth2.Config{
		ClientID:     config.Envs.GoogleKey,
		ClientSecret: config.Envs.GoogleSecret,
		RedirectURL:  "http://locaholhost:8000/auth/google/callback",
		Scopes:       scopes,
		Endpoint:     google.Endpoint,
	}
	return googleService{service: config}
}

func VerifyIdToken(idToken string) (*idtoken.Payload, error) {
	payload, err := idtoken.Validate(context.Background(), idToken, config.Envs.GoogleKey)
	if err != nil {
		return nil, err
	}

	return payload, nil
}
