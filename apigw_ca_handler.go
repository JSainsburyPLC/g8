package g8

import (
	"context"
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/nrlambda"
	"github.com/rs/zerolog"
)

// APIGatewayCustomAuthorizerContext the context for a request for Custom Authorizer
type APIGatewayCustomAuthorizerContext struct {
	Context                    context.Context
	Request                    events.APIGatewayCustomAuthorizerRequestTypeRequest
	Response                   events.APIGatewayCustomAuthorizerResponse
	Logger                     zerolog.Logger
	NewRelicTx                 newrelic.Transaction
	CorrelationID              string
	methodArnParts             methodARN
	hasAtLeastOneAllowedMethod bool
}

// APIGatewayCustomAuthorizerHandlerFunc to populate
type APIGatewayCustomAuthorizerHandlerFunc func(c *APIGatewayCustomAuthorizerContext) error

// APIGatewayCustomAuthorizerHandler fd
func APIGatewayCustomAuthorizerHandler(
	h APIGatewayCustomAuthorizerHandlerFunc,
	conf HandlerConfig,
) func(context.Context, events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error) {

	return func(ctx context.Context, r events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
		if len(r.MethodArn) == 0 {
			return events.APIGatewayCustomAuthorizerResponse{}, errors.New("MethodArn is not set")
		}

		correlationID := getCorrelationIDAPIGW(r.Headers)

		logger := configureLogger(conf).
			Str("route", r.RequestContext.ResourcePath).
			Str("correlation_id", correlationID).
			Str("application", conf.AppName).
			Str("function_name", conf.FunctionName).
			Str("env", conf.EnvName).
			Str("build_version", conf.BuildVersion).
			Logger()

		c := &APIGatewayCustomAuthorizerContext{
			Context:                    ctx,
			Request:                    r,
			Response:                   NewAuthorizerResponse(),
			Logger:                     logger,
			NewRelicTx:                 newrelic.FromContext(ctx),
			CorrelationID:              correlationID,
			methodArnParts:             parseFromMethodARN(r.MethodArn),
			hasAtLeastOneAllowedMethod: false,
		}

		if err := h(c); err != nil {
			logger.Err(err).Msg("Error while calling user-defined function")
			return events.APIGatewayCustomAuthorizerResponse{}, err
		}

		// sanity check
		if !c.hasAtLeastOneAllowedMethod {
			logger.Warn().Msg("Warning! No method were allowed! That means no requests will pass this " +
				"authorizer! Please double check the policy.")
		}
		if len(c.Response.PrincipalID) == 0 {
			logger.Warn().Msg("Warning! The PrincipalID was not defined! Please set it using c.Response.SetPrincipalID() function")
		}

		c.Response.Context = map[string]interface{}{
			"customer-id": c.Response.PrincipalID,
		}

		c.AddNewRelicAttribute("functionName", conf.FunctionName)
		c.AddNewRelicAttribute("route", r.RequestContext.ResourcePath)
		c.AddNewRelicAttribute("correlationID", correlationID)
		c.AddNewRelicAttribute("buildVersion", conf.BuildVersion)

		logger.Debug().
			Str("principal_id", c.Response.PrincipalID).
			Str("account_aws", c.methodArnParts.AccountID).
			Msg("G8 Custom Authorizer successful")

		return c.Response, nil
	}
}

func APIGatewayCustomAuthorizerHandlerWithNewRelic(h APIGatewayCustomAuthorizerHandlerFunc, conf HandlerConfig) lambda.Handler {
	return nrlambda.Wrap(APIGatewayCustomAuthorizerHandler(h, conf), conf.NewRelicApp)
}

func (c *APIGatewayCustomAuthorizerContext) AddNewRelicAttribute(key string, val interface{}) {
	if c.NewRelicTx == nil {
		return
	}
	if err := c.NewRelicTx.AddAttribute(key, val); err != nil {
		c.Logger.Error().Msgf("failed to add attr '%s' to new relic tx: %+v", key, err)
	}
}

func NewAuthorizerResponse() events.APIGatewayCustomAuthorizerResponse {
	return events.APIGatewayCustomAuthorizerResponse{
		PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
			Version: "2012-10-17",
		},
	}
}

func (c *APIGatewayCustomAuthorizerContext) addMethod(effect Effect, verb, resource string) {
	s := events.IAMPolicyStatement{
		Effect:   effect.String(),
		Action:   []string{"execute-api:Invoke"},
		Resource: []string{c.methodArnParts.buildResourceARN(verb, resource)},
	}

	c.Response.PolicyDocument.Statement = append(c.Response.PolicyDocument.Statement, s)
}

func (c *APIGatewayCustomAuthorizerContext) SetPrincipalID(principalID string) {
	c.Response.PrincipalID = principalID
}

func (c *APIGatewayCustomAuthorizerContext) AllowAllMethods() {
	c.hasAtLeastOneAllowedMethod = true
	c.addMethod(Allow, All, "*")
}

func (c *APIGatewayCustomAuthorizerContext) DenyAllMethods() {
	c.addMethod(Deny, All, "*")
}

func (c *APIGatewayCustomAuthorizerContext) AllowMethod(verb, resource string) {
	c.hasAtLeastOneAllowedMethod = true
	c.addMethod(Allow, verb, resource)
}

func (c *APIGatewayCustomAuthorizerContext) DenyMethod(verb, resource string) {
	c.addMethod(Deny, verb, resource)
}
