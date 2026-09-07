package dto

import "whatsnext/backend/internal/model"

type RegisterInput struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type AuthResult struct {
	User         model.User `json:"user"`
	AccessToken  string     `json:"access_token"`
	ExpiresIn    int64      `json:"expires_in"`
	RefreshToken string     `json:"-"`
}
