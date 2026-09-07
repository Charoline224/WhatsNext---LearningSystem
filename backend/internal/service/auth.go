package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	appauth "whatsnext/backend/internal/auth"
	"whatsnext/backend/internal/dto"
	"whatsnext/backend/internal/model"
	"whatsnext/backend/internal/repository"
)

type AuthService struct {
	repo   *repository.AuthRepository
	tokens *appauth.TokenManager
}

func NewAuthService(repo *repository.AuthRepository, tokens *appauth.TokenManager) *AuthService {
	return &AuthService{repo: repo, tokens: tokens}
}

func (s *AuthService) Register(ctx context.Context, input dto.RegisterInput, userAgent string, ip []byte) (dto.AuthResult, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	name := strings.TrimSpace(input.DisplayName)
	if err := validateEmail(email); err != nil {
		return dto.AuthResult{}, err
	}
	if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 80 {
		return dto.AuthResult{}, ValidationError{"display_name", "昵称长度应为 1–80 个字符"}
	}
	if err := validatePassword(input.Password); err != nil {
		return dto.AuthResult{}, err
	}
	hash, err := appauth.HashPassword(input.Password)
	if err != nil {
		return dto.AuthResult{}, err
	}
	user := model.User{ID: newID(), Email: email, DisplayName: name, PasswordHash: hash, Status: "active"}
	plain, token, err := s.newRefresh(user.ID, newID())
	if err != nil {
		return dto.AuthResult{}, err
	}
	if err = s.repo.CreateUserWithToken(ctx, user, token, userAgent, ip); errors.Is(err, repository.ErrConflict) {
		return dto.AuthResult{}, ErrEmailTaken
	} else if err != nil {
		return dto.AuthResult{}, err
	}
	user, err = s.repo.FindUserByID(ctx, user.ID)
	if err != nil {
		return dto.AuthResult{}, err
	}
	return s.authResult(user, plain)
}
func (s *AuthService) Login(ctx context.Context, input dto.LoginInput, userAgent string, ip []byte) (dto.AuthResult, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if validateEmail(email) != nil || input.Password == "" {
		return dto.AuthResult{}, ErrInvalidCredentials
	}
	user, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil || user.Status != "active" || !appauth.VerifyPassword(user.PasswordHash, input.Password) {
		return dto.AuthResult{}, ErrInvalidCredentials
	}
	plain, token, err := s.newRefresh(user.ID, newID())
	if err != nil {
		return dto.AuthResult{}, err
	}
	if err = s.repo.StoreRefreshToken(ctx, token, userAgent, ip); err != nil {
		return dto.AuthResult{}, err
	}
	return s.authResult(user, plain)
}
func (s *AuthService) Refresh(ctx context.Context, oldPlain, userAgent string, ip []byte) (dto.AuthResult, error) {
	if oldPlain == "" {
		return dto.AuthResult{}, ErrUnauthenticated
	}
	plain, replacement, err := s.newRefresh("", "")
	if err != nil {
		return dto.AuthResult{}, err
	}
	user, err := s.repo.RotateRefreshToken(ctx, appauth.HashRefreshToken(oldPlain), replacement, userAgent, ip, time.Now().UTC())
	if err != nil || user.Status != "active" {
		return dto.AuthResult{}, ErrUnauthenticated
	}
	return s.authResult(user, plain)
}
func (s *AuthService) Logout(ctx context.Context, plain string) error {
	if plain == "" {
		return nil
	}
	return s.repo.RevokeRefreshToken(ctx, appauth.HashRefreshToken(plain), time.Now().UTC())
}
func (s *AuthService) GetUser(ctx context.Context, id string) (model.User, error) {
	user, err := s.repo.FindUserByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return model.User{}, ErrUnauthenticated
	}
	if err != nil {
		return model.User{}, err
	}
	if user.Status != "active" {
		return model.User{}, ErrForbidden
	}
	return user, nil
}
func (s *AuthService) newRefresh(userID, familyID string) (string, model.RefreshToken, error) {
	plain, hash, expires, err := s.tokens.NewRefreshToken()
	if err != nil {
		return "", model.RefreshToken{}, err
	}
	if familyID == "" {
		familyID = newID()
	}
	return plain, model.RefreshToken{ID: newID(), UserID: userID, FamilyID: familyID, TokenHash: hash, ExpiresAt: expires}, nil
}
func (s *AuthService) authResult(user model.User, refresh string) (dto.AuthResult, error) {
	access, expires, err := s.tokens.NewAccessToken(user.ID)
	return dto.AuthResult{User: user, AccessToken: access, ExpiresIn: expires, RefreshToken: refresh}, err
}
func validateEmail(value string) error {
	addr, err := mail.ParseAddress(value)
	if err != nil || addr.Address != value || len(value) > 320 {
		return ValidationError{"email", "请输入有效的邮箱地址"}
	}
	return nil
}
func validatePassword(value string) error {
	if len(value) < 8 || len(value) > 128 {
		return ValidationError{"password", "密码长度应为 8–128 个字符"}
	}
	return nil
}
func newID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return id.String()
}
