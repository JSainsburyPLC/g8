package g8

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	"github.com/newrelic/go-agent/v3/integrations/nrlambda"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog"
)

// APIGatewayProxyContext is the context passed to API Gateway proxy handlers.
type APIGatewayProxyContext struct {
	Context       context.Context
	Request       events.APIGatewayProxyRequest
	Response      events.APIGatewayProxyResponse
	Logger        zerolog.Logger
	NewRelicTx    *newrelic.Transaction
	CorrelationID string
}

// APIGatewayProxyHandlerFunc is the user-defined function signature for API Gateway proxy handlers.
type APIGatewayProxyHandlerFunc func(c *APIGatewayProxyContext) error

// APIGatewayProxyHandler is the main handler for API Gateway proxy.
func APIGatewayProxyHandler(
	h APIGatewayProxyHandlerFunc,
	conf HandlerConfig,
) func(context.Context, events.APIGatewayProxyRequest) (any, error) {
	return func(ctx context.Context, r events.APIGatewayProxyRequest) (any, error) {
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

// APIGatewayProxyHandlerWithNewRelic wraps the API Gateway proxy handler with New Relic monitoring.
func APIGatewayProxyHandlerWithNewRelic(h APIGatewayProxyHandlerFunc, conf HandlerConfig) lambda.Handler {
	return nrlambda.Wrap(APIGatewayProxyHandler(h, conf), conf.NewRelicApp)
}

// Bind unmarshals the JSON body of the request into the provided struct and validates it if it implements Validatable.
func (c *APIGatewayProxyContext) Bind(v any) error {
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

// JSON sets the response body to the JSON representation of the provided struct and sets the Content-Type header.
func (c *APIGatewayProxyContext) JSON(statusCode int, body any) error {
	var b []byte
	var err error
	if body != nil {
		b, err = json.Marshal(body)
		if err != nil {
			return err
		}

		if c.Response.Headers == nil {
			c.Response.Headers = make(map[string]string)
		}
		c.Response.Headers["Content-Type"] = "application/json"
	}
	c.Response.StatusCode = statusCode
	c.Response.Body = string(b)
	return nil
}

// AddNewRelicAttribute adds a custom attribute to the New Relic transaction.
func (c *APIGatewayProxyContext) AddNewRelicAttribute(key string, val any) {
	if c.NewRelicTx == nil {
		return
	}
	c.NewRelicTx.AddAttribute(key, val)
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

// GetCookie retrieves the cookie with the given name.
func (c *APIGatewayProxyContext) GetCookie(name string) (http.Cookie, bool) {
	rawCookies := c.GetHeader("cookie")
	if rawCookies == "" {
		return http.Cookie{}, false
	}

	rawRequest := fmt.Sprintf("GET / HTTP/1.0\r\nCookie: %s\r\n\r\n", rawCookies)
	req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(rawRequest)))
	if err != nil {
		return http.Cookie{}, false
	}

	for _, cookie := range req.Cookies() {
		if cookie.Name == name {
			return *cookie, true
		}
	}

	return http.Cookie{}, false
}

// GetHeader retrieves the header value by name. It canonicalizes headers to ensure that values can be accessed
// in a case-insensitive manner.
func (c *APIGatewayProxyContext) GetHeader(name string) string {
	canonicalHeaders := http.Header{}
	for k, v := range c.Request.MultiValueHeaders {
		for _, value := range v {
			canonicalHeaders.Add(k, value)
		}
	}
	return canonicalHeaders.Get(name)
}

func getCorrelationIDAPIGW(headers map[string]string) string {
	correlationID := headers[headerCorrelationID]
	if correlationID != "" {
		return correlationID
	}
	return uuid.New().String()
}
