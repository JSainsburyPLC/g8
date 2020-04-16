package g8_test

import (
	"bytes"
	"errors"
	"net/http"
	"testing"

	adapter "github.com/gaw508/lambda-proxy-http-adapter"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/steinfletcher/apitest"
	"github.com/stretchr/testify/assert"

	"github.com/JSainsburyPLC/g8"
)

func TestError_Error(t *testing.T) {
	err := g8.Err{
		Status: http.StatusBadRequest,
		Code:   "INVALID_QUERY_PARAM",
		Detail: "Invalid query param",
	}

	if err.Error() != "Code: INVALID_QUERY_PARAM; Status: 400; Detail: Invalid query param" {
		t.Fatalf("unexpected error: '%s'", err.Error())
	}
}

func TestAPIGatewayProxyHandler_UnhandledErrorResponseWithStackTrace(t *testing.T) {
	h := func(c *g8.APIGatewayProxyContext) error {
		return eris.Wrap(errors.New("library err"), "application err")
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

	assert.Equal(t, "Unhandled error", jsonPath("$.message", logBuf.Bytes()))
	assert.Equal(t, "library err", jsonPath("$.error.root.message", logBuf.Bytes()))
	assert.NotEmpty(t, jsonPath("$.error.root.stack", logBuf.Bytes()))
	assert.Equal(t, "application err", jsonPath("$.error.wrap[0].message", logBuf.Bytes()))
	assert.NotEmpty(t, jsonPath("$.error.wrap[0].stack", logBuf.Bytes()))
}
