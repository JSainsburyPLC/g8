package g8

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
)

const (
	headerBuildVersion  = "Build-Version"
	headerCorrelationID = "Correlation-Id"
)

// HandlerConfig is the configuration for all handlers.
type HandlerConfig struct {
	AppName      string
	FunctionName string
	EnvName      string
	BuildVersion string
	Logger       zerolog.Logger
	NewRelicApp  *newrelic.Application
}

// LambdaResult is a generic type for Lambda function results.
type LambdaResult any

// StepEvent is a generic type for Step Function events.
type StepEvent any

// Validatable is an interface for types that can be validated.
type Validatable interface {
	Validate() error
}

// Err represents a structured error with status, code, and detail.
type Err struct {
	Status int    `json:"-"`
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// Error implements the error interface for Err.
func (err Err) Error() string {
	errorParts := []string{
		fmt.Sprintf("Code: %s; Status: %d; Detail: %s", err.Code, err.Status, err.Detail),
	}
	return strings.Join(errorParts, "; ")
}

// ToJSON returns the JSON representation of the Err.
var ErrInternalServer = Err{
	Status: http.StatusInternalServerError,
	Code:   "INTERNAL_SERVER_ERROR",
	Detail: "Internal server error",
}

// ErrInvalidBody is returned when the request body is invalid.
var ErrInvalidBody = Err{
	Status: http.StatusBadRequest,
	Code:   "INVALID_REQUEST_BODY",
	Detail: "Invalid request body",
}

// ErrValidation creates a validation error with the given detail message.
func ErrValidation(detail string) Err {
	return Err{
		Status: http.StatusBadRequest,
		Code:   "VALIDATION_ERROR",
		Detail: detail,
	}
}

func configureLogger(conf HandlerConfig) zerolog.Context {
	return conf.Logger.With().
		Str("application", conf.AppName).
		Str("function_name", conf.FunctionName).
		Str("env", conf.EnvName).
		Str("build_version", conf.BuildVersion)
}

func logUnhandledError(logger zerolog.Logger, err error) {
	logger.Error().
		Fields(map[string]any{
			"error": eris.ToJSON(err, true),
		}).
		Msg("Unhandled error")
}
