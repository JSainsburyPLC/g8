package g8

import (
	"fmt"
	"net/http"
	"strings"

	newrelic "github.com/newrelic/go-agent"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
)

const (
	headerBuildVersion  = "Build-Version"
	headerCorrelationID = "Correlation-Id"
)

type HandlerConfig struct {
	AppName      string
	FunctionName string
	EnvName      string
	BuildVersion string
	Logger       zerolog.Logger
	NewRelicApp  newrelic.Application
}

type Validatable interface {
	Validate() error
}

type Err struct {
	Status int    `json:"-"`
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func (err Err) Error() string {
	errorParts := []string{
		fmt.Sprintf("Code: %s; Status: %d; Detail: %s", err.Code, err.Status, err.Detail),
	}
	return strings.Join(errorParts, "; ")
}

var ErrInternalServer = Err{
	Status: http.StatusInternalServerError,
	Code:   "INTERNAL_SERVER_ERROR",
	Detail: "Internal server error",
}

var ErrInvalidBody = Err{
	Status: http.StatusBadRequest,
	Code:   "INVALID_REQUEST_BODY",
	Detail: "Invalid request body",
}

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
	if isErisErr := eris.Unpack(err).ExternalErr == ""; isErisErr {
		logger.Error().
			Fields(map[string]interface{}{
				"error": eris.ToJSON(err, true),
			}).
			Msg("Unhandled error")
	} else {
		logger.Error().Msgf("Unhandled error: %+v", err)
	}
}
