# g8

Provides the following utilities to simplify working with AWS lambda and Api Gateway.

* HTTP request parsing with JSON support and request body validation
* HTTP response writer with JSON support
* Custom error type with JSON support

## Request body parsing

Use the bind method to unmarshal the response body to a struct

```go
type requestBody struct {
	Name   string `json:"name"`
}

handler := func(r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var b requestBody
	err := g8.Bind(r.Body, &b)

	...
}
```

## Request body validation

Implement the `Validate` method on the struct

```go
type requestBody struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func (b body) Validate() error {
	if b.Status == "" {
		return errors.New("status empty")
	}
	return nil
}

handler := func(r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var b requestBody
	err := g8.Bind(r.Body, &b)
	
	// work with requestBody.Name or err.Code == "status empty"
	
	return g8.JSON(http.StatusOK, responseBody)
}
```

## Response writer

There are several methods provided to simplify writing HTTP responses. 

```go
handler := func(r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	...
	g8.JSON(http.StatusOK, responseBody)
}
```

`g8.OK(responseBody)` sets the HTTP status code to `http.StatusOK` and marshals `responseBody` as JSON.

## Errors

### Go Errors

Passing Go errors to the error response writer will log the error and respond with an internal server error

```go
handler := func(r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return g8.Error(errors.New("something went wrong"))
}
```

### Custom Errors

You can pass custom `g8` errors and also map them to HTTP status codes

```go
handler := func(r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return g8.Error(g8.Err{
		Status: http.StatusBadRequest,
		Code:   "SOME_CLIENT_ERROR",
		Detail: "Invalid param",
	})
}
```

Writes the the following response

```json
{
  "code": "SOME_CLIENT_ERROR",
  "detail": "Invalid param"
}
```
