# g8

Provides the following utilities to simplify working with AWS lambda and API Gateway.

* Simple handler interface
* HTTP request parsing with JSON support and request body validation
* HTTP response writer with JSON support
* Custom error type with JSON support
* Logging unhandled errors with a stack trace
* Correlation ID
* New Relic integration

## Request body parsing

Use the bind method to unmarshal the response body to a struct

```go
type requestBody struct {
	Name   string `json:"name"`
}

handler := func(c *g8.APIGatewayProxyContext) error {
	var b requestBody
	err := c.Bind(&b)
	if err != nil {
		return err
	}

	...

	c.JSON(http.StatusOK, responseBody)
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

handler := func(c *g8.APIGatewayProxyContext) error {
	var b requestBody
	err := c.Bind(&b)
	if err != nil {
		// validation error would be returned here
		return err
	}
	
	...
	
	c.JSON(http.StatusOK, responseBody)
}
```

## API Gateway Lambda Authorizer Handlers

You are able to define handlers for [Lambda Authorizer](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-use-lambda-authorizer.html) 
(previously known as custom authorizers) using g8. Here is an example:

```go
    
    handler := g8.APIGatewayCustomAuthorizerHandlerWithNewRelic(
        func(c *APIGatewayCustomAuthorizerContext) error{
            c.Response.SetPrincipalID("some-principal-ID")

            c.Response.AllowAllMethods()
            // other examples:
            // c.Response.DenyAllMethods()
            // c.Response.AllowMethod(Post, "/pets/*")
            return nil
        },
        g8.HandlerConfig{
            ...
        },
    )

    lambda.StartHandler(handler)

```

## Response writing

There are several methods provided to simplify writing HTTP responses. 

```go
handler := func(c *g8.APIGatewayProxyContext) error {
	...
	c.JSON(http.StatusOK, responseBody)
}
```

## Errors

### Go Errors

Returning Go errors to the error response writer will log the error and respond with an internal server error

```go
handler := func(c *g8.APIGatewayProxyContext) error {
	...
	return errors.New("something went wrong")
}
```

### Custom Errors

You can return custom `g8` errors and also map them to HTTP status codes

```go
handler := func(c *g8.APIGatewayProxyContext) error {
	...
	return g8.Err{
		Status: http.StatusBadRequest,
		Code:   "SOME_CLIENT_ERROR",
		Detail: "Invalid param",
	}
}
```

Writes the following response, with status code 400

```json
{
  "code": "SOME_CLIENT_ERROR",
  "detail": "Invalid param"
}
```

### Logging stack traces

Unhandled errors are logged automatically with a stack trace if the error is wrapped by [eris](https://github.com/rotisserie/eris).

The new version of eris handles errors differently and produce a different JSON response compared to the previous version, additionally all errors now produce a JSON response, please fully test these changes. 

```go
eris.Wrapf(err, "failed to send offers to user id: %v", userID)
```

### Pact HTTP provider testing

The `NewHTTPHandler` function can be used to create adaptors for `g8.APIGatewayProxyHandler` lambdas and serve HTTP for pact provider testing to aid engineers and verify that an API provider adheres to a number of pacts authored by its clients.

#### Example
```go
	g8.NewHTTPHandler(LambdaHandlerEndpoints{
		g8.LambdaHandler{
			Handler:    pact.ExampleGetStub,
			Method:     http.MethodGet,
			Path:       "/full/url/path/{var1}/{var2}",
			PathParams: []string{"var1", "var2"},
		},
		g8.LambdaHandler{
			Handler:    pact.ExamplePostStub,
			Method:     http.MethodPost,
			Path:       "/another/full/url/path/{var1}",
			PathParams: []string{"var1"},
		},
	}, 8080)
```

### Requirements
 * Go 1.19+
