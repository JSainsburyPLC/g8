package g8

import (
	"context"
	"strings"

	"github.com/w32blaster/g8/auth"

	"github.com/aws/aws-lambda-go/events"
	newrelic "github.com/newrelic/go-agent"
	"github.com/rs/zerolog"
)

// APIGatewayCustomAuthorizerContext the context for a request for Custom Authorizer
type APIGatewayCustomAuthorizerContext struct {
	Context       context.Context
	Request       events.APIGatewayCustomAuthorizerRequestTypeRequest
	Response      auth.AuthorizerResponse
	Logger        zerolog.Logger
	NewRelicTx    newrelic.Transaction
	CorrelationID string
}

// GetUserPincipalID custom function to find a Pirncipal ID for the current user in a specific way
type GetUserPincipalID func(c *APIGatewayCustomAuthorizerContext) (string, error)

// APIGatewayCustomAuthorizerHandler fd
func APIGatewayCustomAuthorizerHandler(
	getPrincipalID GetUserPincipalID,
	conf HandlerConfig,
) func(context.Context, events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error) {

	return func(ctx context.Context, r events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
		correlationID := getCorrelationIDAPIGW(r.Headers)

		logger := configureLogger(conf).
			Str("route", r.RequestContext.ResourcePath).
			Str("correlation_id", correlationID).
			Logger()

		c := &APIGatewayCustomAuthorizerContext{
			Context:       ctx,
			Request:       r,
			Logger:        logger,
			NewRelicTx:    newrelic.FromContext(ctx),
			CorrelationID: correlationID,
		}

		// DEBUG
		logger.Print("G8 Custom Authorizer: Method ARN: " + r.MethodArn)
		logger.Print(r.Headers)

		// UNIT TEST
		tmp := strings.Split(r.MethodArn, ":")
		apiGatewayArnTmp := strings.Split(tmp[5], "/")
		awsAccountID := tmp[4]

		principalID, err := getPrincipalID(c)
		if err != nil {
			return events.APIGatewayCustomAuthorizerResponse{}, err
		}

		resp := auth.NewAuthorizerResponse(principalID, awsAccountID)
		resp.Region = tmp[3]
		resp.APIID = apiGatewayArnTmp[0]
		resp.Stage = apiGatewayArnTmp[1]

		resp.Context = map[string]interface{}{
			"customer-id": principalID,
		}

		c.AddNewRelicAttribute("functionName", conf.FunctionName)
		c.AddNewRelicAttribute("route", r.RequestContext.ResourcePath)
		c.AddNewRelicAttribute("correlationID", correlationID)
		c.AddNewRelicAttribute("buildVersion", conf.BuildVersion)

		return resp.APIGatewayCustomAuthorizerResponse, nil
	}
}

func (c *APIGatewayCustomAuthorizerContext) AddNewRelicAttribute(key string, val interface{}) {
	if c.NewRelicTx == nil {
		return
	}
	if err := c.NewRelicTx.AddAttribute(key, val); err != nil {
		c.Logger.Error().Msgf("failed to add attr '%s' to new relic tx: %+v", key, err)
	}
}
