package g8

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

type APIGatewayProxyHandler = func(r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

type Validatable interface {
	Validate() error
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

func Bind(data string, v interface{}) error {
	if err := json.Unmarshal([]byte(data), v); err != nil {
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

func Error(err error) (events.APIGatewayProxyResponse, error) {
	var newErr Err
	switch err := err.(type) {
	case Err:
		newErr = err
	default:
		newErr = ErrInternalServer
		fmt.Printf("Unhandled error: %+v", err)
	}

	return JSON(newErr.Status, newErr)
}

func JSON(statusCode int, body interface{}) (events.APIGatewayProxyResponse, error) {
	var b []byte
	var err error
	if body != nil {
		b, err = json.Marshal(body)
		if err != nil {
			statusCode = http.StatusInternalServerError
			b, _ = json.Marshal(ErrInternalServer)
		}
	}

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Body:       string(b),
	}, nil
}

func OK(body interface{}) (events.APIGatewayProxyResponse, error) {
	return JSON(http.StatusOK, body)
}
