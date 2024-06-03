package g8

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	adapter "github.com/jfallis/lambda-proxy-http-adapter"

	"github.com/aws/aws-lambda-go/events"
	"github.com/go-chi/chi/v5"
)

const (
	WelcomeMessage      = "G8 HTTP server is running on port"
	UnhandledErrMessage = "unhandled error: "
)

type LambdaHandlerEndpoints []LambdaHandler

type LambdaHandler struct {
	Handler     any
	Method      string
	PathPattern string
}

// NewHTTPHandler creates a new HTTP server that listens on the given port.
func NewHTTPHandler(lambdaEndpoints LambdaHandlerEndpoints, portNumber int) {
	fmt.Printf("\n%s %d\n\n", WelcomeMessage, portNumber)
	r := chi.NewRouter()
	for _, l := range lambdaEndpoints {
		r.MethodFunc(l.Method, l.PathPattern, LambdaAdapter(l))
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
			fmt.Printf("%s %s \n", r.Method, r.URL.Path)

			buf := new(strings.Builder)
			if _, err := io.Copy(buf, r.Body); err != nil {
				panic(err)
			}

			ctx := &APIGatewayProxyContext{
				Request: adapter.APIGatewayProxyRequestAdaptor(r, buf.String(), l.PathPattern, nil, nil),
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
