package g8_test

import (
	"bytes"
	"errors"
	"fmt"
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

func TestAPIGatewayProxyHandler_UnhandledExternalErrorResponseWithStackTrace(t *testing.T) {
	h := func(c *g8.APIGatewayProxyContext) error {
		return eris.Wrap(errors.New("external error"), "additional context")
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
	assert.Equal(t, "additional context", jsonPath("$.error.root.message", logBuf.Bytes()))
	assert.Equal(t, "external error", jsonPath("$.error.external", logBuf.Bytes()))
	assert.NotEmpty(t, jsonPath("$.error.root.stack", logBuf.Bytes()))
}

// TestAPIGatewayProxyHandler_UnhandledErrorsResponseWithStackTrace tests external errors to json
// https://github.com/rotisserie/eris/blob/a9462968dc6916f50e0c4e89c9d01faa642872f4/format_test.go#L191
func TestAPIGatewayProxyHandler_UnhandledErrorsResponseWithStackTrace(t *testing.T) {
	stackRegex := "g8_test\\.TestAPIGatewayProxyHandler_UnhandledErrorsResponseWithStackTrace:.+:\\d+"
	type rootErr struct {
		message *string
		stack   []string
	}
	type wrapErr struct {
		message string
		stack   string
	}
	type output struct {
		root            rootErr
		wrap            []wrapErr
		externalMessage *string
		message         string
	}

	tests := map[string]struct {
		input error
		output
	}{
		"basic root error": {
			input: eris.New("root error"),
			// {"root":{"message":"root error"}}
			output: output{
				root: rootErr{
					message: strToPtr("root error"),
					stack:   []string{stackRegex},
				},
				message: "Unhandled error",
			},
		},
		"basic wrapped error": {
			input: eris.Wrap(eris.Wrap(eris.New("root error"), "additional context"), "even more context"),
			// {"root":{"message":"root error"},"wrap":[{"message":"even more context"},{"message":"additional context"}]}
			output: output{
				root: rootErr{
					message: strToPtr("root error"),
					stack:   []string{stackRegex, stackRegex, stackRegex},
				},
				wrap: []wrapErr{
					{
						message: "even more context",
						stack:   stackRegex,
					},
					{
						message: "additional context",
						stack:   stackRegex,
					},
				},
				message: "Unhandled error",
			},
		},
		"external error": {
			input: eris.Wrap(errors.New("external error"), "additional context"),
			// {"external":"external error","root":{"message":"additional context"}}
			output: output{
				externalMessage: strToPtr("external error"),
				root: rootErr{
					message: strToPtr("additional context"),
					stack:   []string{stackRegex},
				},
				message: "Unhandled error",
			},
		},
	}
	for desc, tt := range tests {
		t.Run(desc, func(t *testing.T) {
			h := func(c *g8.APIGatewayProxyContext) error {
				return tt.input
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

			// root
			if tt.root.message != nil {
				assert.Equal(t, *tt.root.message, jsonPath("$.error.root.message", logBuf.Bytes()))
			}
			for x, stack := range tt.root.stack {
				assert.Regexp(t, stack, jsonPath(fmt.Sprintf("$.error.root.stack[%d]", x), logBuf.Bytes()))
			}

			// wrap
			for x, stack := range tt.wrap {
				assert.Equal(t, stack.message, jsonPath(fmt.Sprintf("$.error.wrap[%d].message", x), logBuf.Bytes()))
				assert.Regexp(t, stack.stack, jsonPath(fmt.Sprintf("$.error.wrap[%d].stack", x), logBuf.Bytes()))
			}

			// external
			if tt.externalMessage != nil {
				assert.Equal(t, *tt.externalMessage, jsonPath("$.error.external", logBuf.Bytes()))
			}
		})
	}
}

func strToPtr(v string) *string {
	return &v
}
