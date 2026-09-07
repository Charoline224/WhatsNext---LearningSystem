package service

import "errors"

var (
	ErrInvalidInput         = errors.New("invalid input")
	ErrEmailTaken           = errors.New("email already registered")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrUnauthenticated      = errors.New("unauthenticated")
	ErrForbidden            = errors.New("forbidden")
	ErrNotFound             = errors.New("not found")
	ErrFileTooLarge         = errors.New("file too large")
	ErrUnsupportedFile      = errors.New("unsupported file")
	ErrAIUnavailable        = errors.New("AI retrieval unavailable")
	ErrAIProvider           = errors.New("AI provider error")
	ErrRetrievalUnavailable = errors.New("retrieval dependency unavailable")
	ErrConflict             = errors.New("conflict")
)

type ValidationError struct{ Field, Reason string }

func (e ValidationError) Error() string { return e.Field + ": " + e.Reason }
