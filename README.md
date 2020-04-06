# g8

Provides the following utilities to simplify working with AWS lambda and Api Gateway.

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

## Authorization Request

You can define your custom authorization logic in the *AWS API Gateway*, where you can implement how your users will be 
authorized. This is done my defining ["Custom Authorizer"](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-use-lambda-authorizer.html) 
inside your API Gateway, that can call a lambda function. But 
in this case the request and response of this lambda function should follow defined structure. In order to support custom 
authorization lambda function G8 framework offers you some ready integration.

```go

    func getUserDetailsHandler() g8.GetUserPincipalID {
        // function where you define how to t
        return func(c *g8.APIGatewayCustomAuthorizerContext) (string, error) {
            return "some-user-principal-id", nil
        }
    }
    
    ...

	handler := g8.APIGatewayCustomAuthorizerHandlerWithNewRelic(
		getUserDetailsHandler(),
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

Writes the the following response, with status code 400

```json
{
  "code": "SOME_CLIENT_ERROR",
  "detail": "Invalid param"
}
```

### Logging stack traces

Unhandled errors are logged automatically with a stack trace if the error is wrapped by [eris](https://github.com/rotisserie/eris). 

```go
eris.Wrapf(err, "failed to send offers to user id: %v", userID)
```
