package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"whatsnext/backend/internal/model"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type AuthRepository struct{ db *sqlx.DB }

func NewAuthRepository(db *sqlx.DB) *AuthRepository { return &AuthRepository{db: db} }

func (r *AuthRepository) CreateUserWithToken(ctx context.Context, user model.User, token model.RefreshToken, userAgent string, ip []byte) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin register transaction: %w", err)
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO users (id,email,display_name,password_hash,status) VALUES (?,?,?,?,?)`, user.ID, user.Email, user.DisplayName, user.PasswordHash, user.Status)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return ErrConflict
		}
		return fmt.Errorf("insert user: %w", err)
	}
	if err := insertRefreshToken(ctx, tx, token, userAgent, ip); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit register transaction: %w", err)
	}
	return nil
}

func (r *AuthRepository) FindUserByEmail(ctx context.Context, email string) (model.User, error) {
	var user model.User
	err := r.db.GetContext(ctx, &user, `SELECT id,email,display_name,password_hash,status,created_at,updated_at FROM users WHERE email=?`, email)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, ErrNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("find user by email: %w", err)
	}
	return user, nil
}
func (r *AuthRepository) FindUserByID(ctx context.Context, id string) (model.User, error) {
	var user model.User
	err := r.db.GetContext(ctx, &user, `SELECT id,email,display_name,password_hash,status,created_at,updated_at FROM users WHERE id=?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, ErrNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("find user by id: %w", err)
	}
	return user, nil
}
func (r *AuthRepository) StoreRefreshToken(ctx context.Context, token model.RefreshToken, userAgent string, ip []byte) error {
	return insertRefreshToken(ctx, r.db, token, userAgent, ip)
}

func (r *AuthRepository) RotateRefreshToken(ctx context.Context, oldHash []byte, replacement model.RefreshToken, userAgent string, ip []byte, now time.Time) (model.User, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return model.User{}, fmt.Errorf("begin refresh transaction: %w", err)
	}
	defer tx.Rollback()
	var old model.RefreshToken
	err = tx.GetContext(ctx, &old, `SELECT id,user_id,family_id,token_hash,expires_at,revoked_at FROM refresh_tokens WHERE token_hash=? FOR UPDATE`, oldHash)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, ErrInvalidRefreshToken
	}
	if err != nil {
		return model.User{}, fmt.Errorf("select refresh token: %w", err)
	}
	if old.RevokedAt != nil {
		_, _ = tx.ExecContext(ctx, `UPDATE refresh_tokens SET revoked_at=COALESCE(revoked_at,?) WHERE family_id=?`, now, old.FamilyID)
		_ = tx.Commit()
		return model.User{}, ErrRefreshTokenReused
	}
	if !old.ExpiresAt.After(now) {
		return model.User{}, ErrInvalidRefreshToken
	}
	replacement.UserID = old.UserID
	replacement.FamilyID = old.FamilyID
	if err := insertRefreshToken(ctx, tx, replacement, userAgent, ip); err != nil {
		return model.User{}, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE refresh_tokens SET revoked_at=?,replaced_by_token_id=? WHERE id=? AND revoked_at IS NULL`, now, replacement.ID, old.ID)
	if err != nil {
		return model.User{}, fmt.Errorf("revoke refresh token: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return model.User{}, ErrInvalidRefreshToken
	}
	var user model.User
	err = tx.GetContext(ctx, &user, `SELECT id,email,display_name,password_hash,status,created_at,updated_at FROM users WHERE id=?`, old.UserID)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, ErrNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("select refresh user: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return model.User{}, fmt.Errorf("commit refresh transaction: %w", err)
	}
	return user, nil
}

func (r *AuthRepository) RevokeRefreshToken(ctx context.Context, hash []byte, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE refresh_tokens SET revoked_at=COALESCE(revoked_at,?) WHERE token_hash=?`, now, hash)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

type sqlExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func insertRefreshToken(ctx context.Context, exec sqlExecer, token model.RefreshToken, userAgent string, ip []byte) error {
	_, err := exec.ExecContext(ctx, `INSERT INTO refresh_tokens (id,user_id,family_id,token_hash,expires_at,user_agent,ip_address) VALUES (?,?,?,?,?,?,?)`, token.ID, token.UserID, token.FamilyID, token.TokenHash, token.ExpiresAt, nullableString(userAgent), nullableBytes(ip))
	if err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}
func nullableString(v string) any {
	if v == "" {
		return nil
	}
	return v
}
func nullableBytes(v []byte) any {
	if len(v) == 0 {
		return nil
	}
	return v
}
