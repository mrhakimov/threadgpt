package domain

import (
	"errors"
	"net/http"
)

var (
	ErrInvalidArgument     = errors.New("invalid argument")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden")
	ErrNotFound            = errors.New("not found")
	ErrRateLimited         = errors.New("rate limited")
	ErrInvalidAPIKey       = errors.New("invalid api key")
	ErrQuotaExceeded       = errors.New("quota exceeded")
	ErrProviderUnavailable = errors.New("provider unavailable")
	ErrInternal            = errors.New("internal error")
)

type ErrorDescriptor struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func DescribeError(err error) ErrorDescriptor {
	switch {
	case errors.Is(err, ErrInvalidArgument):
		return ErrorDescriptor{
			Code:    "invalid_request",
			Message: "The request was invalid.",
			Status:  http.StatusBadRequest,
		}
	case errors.Is(err, ErrInvalidAPIKey):
		return ErrorDescriptor{
			Code:    "invalid_api_key",
			Message: "OpenAI rejected this API key. Check it and try again.",
			Status:  http.StatusUnauthorized,
		}
	case errors.Is(err, ErrUnauthorized):
		return ErrorDescriptor{
			Code:    "unauthorized",
			Message: "Your session has expired. Please sign in again.",
			Status:  http.StatusUnauthorized,
		}
	case errors.Is(err, ErrForbidden):
		return ErrorDescriptor{
			Code:    "forbidden",
			Message: "You do not have access to this resource.",
			Status:  http.StatusForbidden,
		}
	case errors.Is(err, ErrNotFound):
		return ErrorDescriptor{
			Code:    "not_found",
			Message: "That resource was not found.",
			Status:  http.StatusNotFound,
		}
	case errors.Is(err, ErrQuotaExceeded):
		return ErrorDescriptor{
			Code:    "quota_exceeded",
			Message: "This OpenAI API key has run out of quota. Check your usage and billing, then try again.",
			Status:  http.StatusTooManyRequests,
		}
	case errors.Is(err, ErrRateLimited):
		return ErrorDescriptor{
			Code:    "rate_limited",
			Message: "Too many requests. Please wait a moment and try again.",
			Status:  http.StatusTooManyRequests,
		}
	case errors.Is(err, ErrProviderUnavailable):
		return ErrorDescriptor{
			Code:    "server_error",
			Message: "OpenAI is unavailable right now. Please try again in a moment.",
			Status:  http.StatusBadGateway,
		}
	default:
		return ErrorDescriptor{
			Code:    "server_error",
			Message: "Something went wrong on the server. Please try again.",
			Status:  http.StatusInternalServerError,
		}
	}
}
