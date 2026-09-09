package domain

import (
	"errors"
	"net/http"
)

type AppError struct {
	Code    int
	Message string
	Detail  string
}

func (e *AppError) Error() string { return e.Message }

var (
	ErrNotFound           = &AppError{Code: http.StatusNotFound, Message: "not found"}
	ErrConflict           = &AppError{Code: http.StatusConflict, Message: "already exists"}
	ErrValidation         = &AppError{Code: http.StatusUnprocessableEntity, Message: "validation failed"}
	ErrForbidden          = &AppError{Code: http.StatusForbidden, Message: "forbidden"}
	ErrUnauthorized       = &AppError{Code: http.StatusUnauthorized, Message: "unauthorized"}
	ErrInvalidCredentials = &AppError{Code: http.StatusUnauthorized, Message: "invalid credentials"}
	ErrUserAlreadyExists  = &AppError{Code: http.StatusConflict, Message: "user already exists"}
	ErrNotExistEmail      = &AppError{Code: http.StatusNotFound, Message: "email does not exist"}
	ErrDomainNotAllowed   = &AppError{Code: http.StatusForbidden, Message: "domain not allowed"}
)

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var appErr *AppError
	return errors.Is(err, ErrNotFound) || (errors.As(err, &appErr) && appErr.Code == http.StatusNotFound)
}
