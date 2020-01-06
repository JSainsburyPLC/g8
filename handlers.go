package g8

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/nrlambda"
	"github.com/rs/zerolog"
)

type HandlerConfig struct {
	Logger      zerolog.Logger
	NewRelicApp newrelic.Application
}

type APIGatewayProxyContext struct {
	Request    events.APIGatewayProxyRequest
	Response   events.APIGatewayProxyResponse
	Logger     zerolog.Logger
	NewRelicTx newrelic.Transaction
}

type APIGatewayProxyHandlerFunc func(c *APIGatewayProxyContext) error

func APIGatewayProxyHandler(h APIGatewayProxyHandlerFunc, conf HandlerConfig) lambda.Handler {
	return nrlambda.Wrap(func(r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		logger := conf.Logger.With().
			Str("ContextKey", "ContextValue"). // TODO: log context values
			Logger()

		c := &APIGatewayProxyContext{
			Request:    r,
			Logger:     logger,

			// TODO: Need to figure out how to get tx. Doesn't seem to be an easy way to do this.
			//NewRelicTx: "",
		}

		err := h(c)
		if err != nil {
			c.handleError(err)
			return c.Response, nil
		}

		return c.Response, nil
	}, conf.NewRelicApp)
}

func (c *APIGatewayProxyContext) Bind(v interface{}) error {
	if err := json.Unmarshal([]byte(c.Request.Body), v); err != nil {
		return ErrInvalidBody
	}

	if validatable, ok := v.(Validatable); ok {
		err := validatable.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *APIGatewayProxyContext) JSON(statusCode int, body interface{}) error {
	var b []byte
	var err error
	if body != nil {
		b, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	c.Response.StatusCode = statusCode
	c.Response.Body = string(b)
	return nil
}

func (c *APIGatewayProxyContext) handleError(err error) {
	var newErr Err
	switch err := err.(type) {
	case Err:
		newErr = err
	default:
		newErr = ErrInternalServer
		c.Logger.Error().Msgf("Unhandled error: %+v", err)
	}
	_ = c.JSON(newErr.Status, newErr)
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
