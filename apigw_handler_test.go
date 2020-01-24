package g8_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/JSainsburyPLC/g8"

	"github.com/PaesslerAG/jsonpath"
	"github.com/aws/aws-lambda-go/events"
	adapter "github.com/gaw508/lambda-proxy-http-adapter"
	"github.com/rs/zerolog"
	"github.com/steinfletcher/apitest"
	"github.com/stretchr/testify/assert"
)

type body struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func (b body) Validate() error {
	if b.Status == "" {
		return errors.New("status empty")
	}
	return nil
}

func TestAPIGatewayProxyContext_Bind(t *testing.T) {
	testCases := map[string]struct {
		c            *g8.APIGatewayProxyContext
		expectedBody body
		expectedErr  error
	}{
		"success": {
			c: &g8.APIGatewayProxyContext{
				Request: events.APIGatewayProxyRequest{
					Body: `{"name":"one","status":"ok"}`,
				},
			},
			expectedBody: body{
				Name:   "one",
				Status: "ok",
			},
			expectedErr: nil,
		},
		"invalid json": {
			c: &g8.APIGatewayProxyContext{
				Request: events.APIGatewayProxyRequest{
					Body: `NOTJSON`,
				},
			},
			expectedBody: body{},
			expectedErr: g8.Err{
				Status: 400,
				Code:   "INVALID_REQUEST_BODY",
				Detail: "Invalid request body",
			},
		},
		"validation error": {
			c: &g8.APIGatewayProxyContext{
				Request: events.APIGatewayProxyRequest{
					Body: `{"name":"two"}`,
				},
			},
			expectedBody: body{
				Name: "two",
			},
			expectedErr: errors.New("status empty"),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			var b body
			err := tc.c.Bind(&b)

			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedBody, b)
		})
	}
}

func TestAPIGatewayProxyContext_JSON(t *testing.T) {
	testCases := map[string]struct {
		c           *g8.APIGatewayProxyContext
		statusCode  int
		body        interface{}
		expectedRes events.APIGatewayProxyResponse
	}{
		"no body": {
			c:          &g8.APIGatewayProxyContext{},
			statusCode: 202,
			body:       nil,
			expectedRes: events.APIGatewayProxyResponse{
				StatusCode: 202,
				Body:       "",
			},
		},
		"body": {
			c:          &g8.APIGatewayProxyContext{},
			statusCode: 200,
			body: body{
				Name:   "one",
				Status: "ok",
			},
			expectedRes: events.APIGatewayProxyResponse{
				StatusCode: 200,
				Body:       `{"name":"one","status":"ok"}`,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := tc.c.JSON(tc.statusCode, tc.body)

			assert.Nil(t, err)
			assert.Equal(t, tc.expectedRes, tc.c.Response)
		})
	}
}

func TestAPIGatewayProxyHandler_SuccessResponse(t *testing.T) {
	h := func(c *g8.APIGatewayProxyContext) error {
		var b body
		err := c.Bind(&b)
		if err != nil {
			return err
		}

		c.Logger.Info().Msg("testing logger")

		return c.JSON(http.StatusOK, map[string]string{
			"one":   "two",
			"three": "four",
		})
	}

	logBuf := &bytes.Buffer{}
	lh := g8.APIGatewayProxyHandler(h, g8.HandlerConfig{
		Logger: zerolog.New(logBuf),
	})

	apitest.New().
		Handler(adapter.GetHttpHandlerWithContext(lh, "/", nil)).
		Post("/").
		JSON(`{
					"name": "one",
					"status": "ok"
				}`).
		Expect(t).
		Status(http.StatusOK).
		Body(`{
					"one": "two",
					"three": "four"
				}`).
		HeaderPresent("Correlation-Id").
		End()

	assert.True(t, containsLogMessage(logBuf.String(), "testing logger"))
}

func TestAPIGatewayProxyHandler_G8ErrorResponse(t *testing.T) {
	h := func(c *g8.APIGatewayProxyContext) error {
		return g8.Err{
			Status: 401,
			Code:   "UNAUTHORIZED",
			Detail: "Unauthorized",
		}
	}

	lh := g8.APIGatewayProxyHandler(h, g8.HandlerConfig{})

	apitest.New().
		Handler(adapter.GetHttpHandlerWithContext(lh, "/", nil)).
		Get("/").
		Expect(t).
		Status(http.StatusUnauthorized).
		Body(`{
					"code": "UNAUTHORIZED",
					"detail": "Unauthorized"
				}`).
		HeaderPresent("Correlation-Id").
		End()
}

func TestAPIGatewayProxyHandler_UnhandledErrorResponse(t *testing.T) {
	h := func(c *g8.APIGatewayProxyContext) error {
		return errors.New("some error")
	}

	logBuf := &bytes.Buffer{}
	lh := g8.APIGatewayProxyHandler(h, g8.HandlerConfig{
		Logger: zerolog.New(logBuf),
	})

	apitest.New().
		Handler(adapter.GetHttpHandlerWithContext(lh, "/", nil)).
		Get("/").
		Expect(t).
		Status(http.StatusInternalServerError).
		Body(`{
					"code": "INTERNAL_SERVER_ERROR",
					"detail": "Internal server error"
				}`).
		HeaderPresent("Correlation-Id").
		End()

	assert.Equal(t, "Unhandled error: some error", jsonPath("$.message", logBuf.Bytes()))
}

func containsLogMessage(fullLog string, message string) bool {
	type log struct {
		Message string `json:"message"`
	}

	lines := strings.Split(fullLog, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var parsedLine log
		_ = json.Unmarshal([]byte(line), &parsedLine)
		if parsedLine.Message == message {
			return true
		}
	}
	return false
}

func jsonPath(path string, jsonData []byte) interface{} {
	v := interface{}(nil)
	err := json.Unmarshal(jsonData, &v)
	if err != nil {
		panic(err)
	}

	value, err := jsonpath.Get(path, v)
	if err != nil {
		panic(err)
	}
	return value
}
