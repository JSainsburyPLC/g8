package g8

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/nrlambda"
	"github.com/rs/zerolog"
)

type APIGatewayProxyContext struct {
	Context       context.Context
	Request       events.APIGatewayProxyRequest
	Response      events.APIGatewayProxyResponse
	Logger        zerolog.Logger
	NewRelicTx    newrelic.Transaction
	CorrelationID string
}

type APIGatewayProxyHandlerFunc func(c *APIGatewayProxyContext) error

func APIGatewayProxyHandler(
	h APIGatewayProxyHandlerFunc,
	conf HandlerConfig,
) func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return func(ctx context.Context, r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		correlationID := getCorrelationIDAPIGW(r.Headers)

		logger := configureLogger(conf).
			Str("route", r.RequestContext.ResourcePath).
			Str("correlation_id", correlationID).
			Logger()

		c := &APIGatewayProxyContext{
			Context:       ctx,
			Request:       r,
			Logger:        logger,
			NewRelicTx:    newrelic.FromContext(ctx),
			CorrelationID: correlationID,
		}

		if c.Response.Headers == nil {
			c.Response.Headers = make(map[string]string)
		}
		c.Response.Headers[headerCorrelationID] = correlationID
		c.Response.Headers[headerBuildVersion] = conf.BuildVersion

		c.AddNewRelicAttribute("functionName", conf.FunctionName)
		c.AddNewRelicAttribute("route", r.RequestContext.ResourcePath)
		c.AddNewRelicAttribute("correlationID", correlationID)
		c.AddNewRelicAttribute("buildVersion", conf.BuildVersion)

		err := h(c)
		if err != nil {
			c.handleError(err)
			return c.Response, nil
		}

		return c.Response, nil
	}
}

func APIGatewayProxyHandlerWithNewRelic(h APIGatewayProxyHandlerFunc, conf HandlerConfig) lambda.Handler {
	return nrlambda.Wrap(APIGatewayProxyHandler(h, conf), conf.NewRelicApp)
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

func (c *APIGatewayProxyContext) AddNewRelicAttribute(key string, val interface{}) {
	if c.NewRelicTx == nil {
		return
	}
	if err := c.NewRelicTx.AddAttribute(key, val); err != nil {
		c.Logger.Error().Msgf("failed to add attr '%s' to new relic tx: %+v", key, err)
	}
}

func (c *APIGatewayProxyContext) handleError(err error) {
	var newErr Err
	switch err := err.(type) {
	case Err:
		newErr = err
	default:
		newErr = ErrInternalServer
		logUnhandledError(c.Logger, err)
	}
	_ = c.JSON(newErr.Status, newErr)
}

func getCorrelationIDAPIGW(headers map[string]string) string {
	correlationID := headers[headerCorrelationID]
	if correlationID != "" {
		return correlationID
	}
	return uuid.New().String()
}
