package g8

import (
	"context"
	"errors"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/newrelic/go-agent/_integrations/nrlambda"
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

// GetUserPrincipalID custom function to find a Pirncipal ID for the current user in a specific way
type GetUserPrincipalID func(c *APIGatewayCustomAuthorizerContext) (string, error)

// ApplyMethodRules is the function where calling code may declare rules for methods
type ApplyMethodRules func(r *auth.AuthorizerResponse)

// APIGatewayCustomAuthorizerHandler fd
func APIGatewayCustomAuthorizerHandler(
	fnGetPrincipalID GetUserPrincipalID,
	fnApplyMethodRules ApplyMethodRules,
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

		if len(r.MethodArn) == 0 {
			return events.APIGatewayCustomAuthorizerResponse{}, errors.New("MethodArn is not set")
		}

		// DEBUG
		logger.Info().Str("method_arn", r.MethodArn).Msg("G8 Custom Authorizer")

		// UNIT TEST
		tmp := strings.Split(r.MethodArn, ":")
		apiGatewayArnTmp := strings.Split(tmp[5], "/")
		awsAccountID := tmp[4]

		principalID, err := fnGetPrincipalID(c)
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

		fnApplyMethodRules(resp)

		// sanity check
		if !resp.HasAllowingMethod() {
			logger.Warn().Msg("Warning! No method were allowed! That means no requests will pass this " +
				"authorizer! Please double check the policy.")
		}

		c.AddNewRelicAttribute("functionName", conf.FunctionName)
		c.AddNewRelicAttribute("route", r.RequestContext.ResourcePath)
		c.AddNewRelicAttribute("correlationID", correlationID)
		c.AddNewRelicAttribute("buildVersion", conf.BuildVersion)

		logger.Info().
			Str("function_name", conf.FunctionName).
			Str("route", r.RequestContext.ResourcePath).
			Str("principal_id", principalID).
			Str("account_aws", awsAccountID).
			Msg("G8 Custom Authorizer successful")

		return resp.APIGatewayCustomAuthorizerResponse, nil
	}
}

func APIGatewayCustomAuthorizerHandlerWithNewRelic(h GetUserPrincipalID, r ApplyMethodRules, conf HandlerConfig) lambda.Handler {
	return nrlambda.Wrap(APIGatewayCustomAuthorizerHandler(h, r, conf), conf.NewRelicApp)
}

func (c *APIGatewayCustomAuthorizerContext) AddNewRelicAttribute(key string, val interface{}) {
	if c.NewRelicTx == nil {
		return
	}
	if err := c.NewRelicTx.AddAttribute(key, val); err != nil {
		c.Logger.Error().Msgf("failed to add attr '%s' to new relic tx: %+v", key, err)
	}
}
