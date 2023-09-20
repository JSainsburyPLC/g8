package g8

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/go-chi/chi/v5"
)

const (
	WelcomeMessage      = "G8 HTTP server is running on port"
	UnhandledErrMessage = "unhandled error: "
)

type LambdaHandlerEndpoints []LambdaHandler

type LambdaHandler struct {
	Handler    any
	Method     string
	Path       string
	PathParams []string
}

// NewHTTPHandler creates a new HTTP server that listens on the given port.
func NewHTTPHandler(lambdaEndpoints LambdaHandlerEndpoints, portNumber int) {
	fmt.Printf("\n%s %d\n\n", WelcomeMessage, portNumber)
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
		case func(ctx *APIGatewayProxyContext) error:
			fmt.Printf("%s %s: %+v \n", r.Method, r.URL.Path, l.PathParams)
			ctx := &APIGatewayProxyContext{
				Request: NewAPIGatewayRequestBuilder(r, l.PathParams).Request(),
			}

			if eErr := eventHandler(ctx); eErr != nil {
				fmt.Printf("%s %s\n", UnhandledErrMessage, eErr.Error())
				resp, uErr := unhandledError(eErr)
				if uErr != nil {
					panic(uErr)
				}
				ctx.Response = resp
			}

			w.Header().Set("Content-Type", "application/json")
			for k, v := range ctx.Response.Headers {
				w.Header().Set(k, v)
			}
			for k, v := range ctx.Response.MultiValueHeaders {
				w.Header().Set(k, strings.Join(v, ","))
			}
			w.WriteHeader(ctx.Response.StatusCode)
			if _, wErr := w.Write([]byte(ctx.Response.Body)); wErr != nil {
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
func unhandledError(err error) (events.APIGatewayProxyResponse, error) {
	var newErr Err
	switch err := err.(type) {
	case Err:
		newErr = err
	default:
		newErr = ErrInternalServer
	}

	var r events.APIGatewayProxyResponse
	b, err := json.Marshal(newErr)
	if err != nil {
		return r, err
	}

	r.Headers = make(map[string]string)
	r.Headers["Content-Type"] = "application/json"
	r.StatusCode = newErr.Status
	r.Body = string(b)

	return r, nil
}
