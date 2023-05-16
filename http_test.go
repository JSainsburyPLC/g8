package g8_test

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JSainsburyPLC/g8"
)

func TestLambdaAdapter(t *testing.T) {
	l := g8.LambdaHandler{
		Handler: func(ctx context.Context, r events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
			return &events.APIGatewayProxyResponse{
				StatusCode:        http.StatusOK,
				Headers:           map[string]string{"Content-Type": "text/plain"},
				MultiValueHeaders: map[string][]string{"Set-Cookie": {"cookie1", "cookie2"}},
				Body:              "success",
			}, nil
		},
		Method:     http.MethodGet,
		Path:       "/test/url/path/{var1}/{var2}",
		PathParams: []string{"var1", "var2"},
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test/url/path/var1/var2", nil)
	r.Header.Set("Content-Type", "text/plain")
	g8.LambdaAdapter(l)(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain", w.Header().Get("Content-Type"))
	assert.Equal(t, "cookie1,cookie2", w.Header().Get("Set-Cookie"))
	assert.Equal(t, "success", w.Body.String())
}

func TestLambdaAdapter_without_content_type(t *testing.T) {
	l := g8.LambdaHandler{
		Handler: func(ctx context.Context, r events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
			return &events.APIGatewayProxyResponse{
				StatusCode:        http.StatusOK,
				MultiValueHeaders: map[string][]string{"Set-Cookie": {"cookie1", "cookie2"}},
				Body:              `{"message":"success"}`,
			}, nil
		},
		Method:     http.MethodGet,
		Path:       "/test/url/path/{var1}/{var2}",
		PathParams: []string{"var1", "var2"},
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test/url/path/var1/var2", nil)
	g8.LambdaAdapter(l)(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "cookie1,cookie2", w.Header().Get("Set-Cookie"))
	assert.Equal(t, `{"message":"success"}`, w.Body.String())
}

func TestLambdaAdapter_g8_error(t *testing.T) {
	l := g8.LambdaHandler{
		Handler: func(ctx context.Context, r events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
			return nil, g8.ErrInternalServer
		},
		Method: http.MethodGet,
		Path:   "/test/url/path",
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test/url/path/var1/var2", nil)
	r.Header.Set("Content-Type", "application/json")
	g8.LambdaAdapter(l)(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, `{"code":"INTERNAL_SERVER_ERROR","detail":"Internal server error"}`, w.Body.String())
}

func TestLambdaAdapter_generic_error(t *testing.T) {
	l := g8.LambdaHandler{
		Handler: func(ctx context.Context, r events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
			return nil, fmt.Errorf("generic error")
		},
		Method: http.MethodGet,
		Path:   "/test/url/path",
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test/url/path/var1/var2", nil)
	r.Header.Set("Content-Type", "application/json")
	g8.LambdaAdapter(l)(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, `{"code":"INTERNAL_SERVER_ERROR","detail":"Internal server error"}`, w.Body.String())
}
