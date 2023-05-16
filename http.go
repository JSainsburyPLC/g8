package g8

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"strings"
)

const (
	WELCOME_MESSAGE       = "G8 HTTP server is running on port"
	UNHANDLED_ERR_MESSAGE = "unhandled error: "
)

type LambdaHandlerEndpoints []LambdaHandler

type LambdaHandler struct {
	Handler    interface{}
	Method     string
	Path       string
	PathParams []string
}

// NewHTTPHandler creates a new HTTP server that listens on the given port.
func NewHTTPHandler(lambdaEndpoints LambdaHandlerEndpoints, portNumber int) {
	fmt.Printf("\n%s %d\n\n", WELCOME_MESSAGE, portNumber)
	r := chi.NewRouter()
	for _, l := range lambdaEndpoints {
		r.MethodFunc(l.Method, l.Path, LambdaAdapter(l))
	}
	if err := http.ListenAndServe(fmt.Sprintf(":%d", portNumber), r); err != nil {
		panic(err)
	}
}

// LambdaAdapter converts a LambdaHandler into a http.HandlerFunc.
func LambdaAdapter(l LambdaHandler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch eventHandler := l.Handler.(type) {
		// APIGatewayProxyHandler
		case func(context.Context, events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error):
			fmt.Printf("%s %s: %+v \n", r.Method, r.URL.Path, l.PathParams)
			request := NewAPIGatewayRequestBuilder(r, l.PathParams)
			resp, eErr := eventHandler(context.Background(), request.Request())
			if eErr != nil {
				fmt.Printf("%s %s\n", UNHANDLED_ERR_MESSAGE, eErr.Error())
				resp, eErr = unhandledError(eErr)
				if eErr != nil {
					panic(eErr)
				}
			}

			if resp.Headers == nil {
				resp.Headers = make(map[string]string)
			}
			resp.Headers["Content-Type"] = "application/json"
			w.WriteHeader(resp.StatusCode)
			for k, v := range resp.Headers {
				w.Header().Set(k, v)
			}
			for k, v := range resp.MultiValueHeaders {
				w.Header().Set(k, strings.Join(v, ","))
			}

			if _, wErr := w.Write([]byte(resp.Body)); wErr != nil {
				panic(wErr)
			}
		default:
			panic(fmt.Sprintf("unknown type: %T", l.Handler))
		}
	}
}

// APIGatewayRequestBuilder is a builder for APIGatewayProxyRequest.
type APIGatewayRequestBuilder struct {
	pathParams []string
	request    *http.Request
}

// Headers returns the headers of the request.
func (b *APIGatewayRequestBuilder) Headers() map[string]string {
	headers := make(map[string]string, len(b.request.Header))
	for k, v := range b.request.Header {
		headers[k] = strings.Join(v, ",")
	}

	return headers
}

// QueryStrings returns the query strings of the request.
func (b *APIGatewayRequestBuilder) QueryStrings() (map[string]string, map[string][]string) {
	query := b.request.URL.Query()
	queryParams := make(map[string]string, len(query))
	MultiQueryParams := make(map[string][]string, len(query))
	for k, v := range query {
		queryParams[k] = strings.Join(v, ",")
		MultiQueryParams[k] = v
	}

	return queryParams, MultiQueryParams
}

// PathParams returns the path parameters of the request.
func (b *APIGatewayRequestBuilder) PathParams() map[string]string {
	pathParams := make(map[string]string, len(b.pathParams))
	for _, v := range b.pathParams {
		pathParams[v] = chi.URLParam(b.request, v)
	}

	return pathParams
}

// Body returns the body of the request.
func (b *APIGatewayRequestBuilder) Body() string {
	if body, err := io.ReadAll(b.request.Body); err == nil {
		return string(body)
	}
	return ""
}

// Request returns the APIGatewayProxyRequest.
func (b *APIGatewayRequestBuilder) Request() events.APIGatewayProxyRequest {
	query, multiQuery := b.QueryStrings()
	return events.APIGatewayProxyRequest{
		Path:                            b.request.URL.Path,
		HTTPMethod:                      b.request.Method,
		Headers:                         b.Headers(),
		MultiValueHeaders:               b.request.Header,
		QueryStringParameters:           query,
		MultiValueQueryStringParameters: multiQuery,
		PathParameters:                  b.PathParams(),
		Body:                            b.Body(),
	}
}

// NewAPIGatewayRequestBuilder creates a new APIGatewayRequestBuilder.
func NewAPIGatewayRequestBuilder(request *http.Request, pathParams []string) *APIGatewayRequestBuilder {
	return &APIGatewayRequestBuilder{
		request:    request,
		pathParams: pathParams,
	}
}

// unhandledError returns an APIGatewayProxyResponse with the given error.
func unhandledError(err error) (*events.APIGatewayProxyResponse, error) {
	var newErr Err
	switch err := err.(type) {
	case Err:
		newErr = err
	default:
		newErr = ErrInternalServer
	}

	b, err := json.Marshal(newErr)
	if err != nil {
		return nil, err
	}

	r := new(events.APIGatewayProxyResponse)
	r.Headers = make(map[string]string)
	r.Headers["Content-Type"] = "application/json"
	r.StatusCode = newErr.Status
	r.Body = string(b)

	return r, nil
}
