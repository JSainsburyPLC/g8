package g8

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	"github.com/newrelic/go-agent/v3/integrations/nrlambda"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog"
)

// CloudWatchContext is the context passed to CloudWatch event handlers.
type CloudWatchContext struct {
	Context       context.Context
	Event         events.CloudWatchEvent
	Logger        zerolog.Logger
	NewRelicTx    *newrelic.Transaction
	CorrelationID string
}

// CloudWatchHandlerFunc is the user-defined function signature for CloudWatch event handlers.
type CloudWatchHandlerFunc func(c *CloudWatchContext) (LambdaResult, error)

// CloudWatchHandler is the main handler for CloudWatch events.
func CloudWatchHandler(
	h CloudWatchHandlerFunc,
	conf HandlerConfig,
) func(context.Context, events.CloudWatchEvent) (LambdaResult, error) {
	return func(ctx context.Context, event events.CloudWatchEvent) (LambdaResult, error) {
		correlationID := uuid.New().String()

		// the resource that triggered the event, e.g. "arn:aws:events:us-east-1:123456789012:rule/MyScheduledRule"
		var cloudWatchResource string
		if len(event.Resources) > 0 {
			cloudWatchResource = event.Resources[0]
		}

		logger := configureLogger(conf).
			Str("cloud_watch_resource", cloudWatchResource).
			Str("correlation_id", correlationID).
			Logger()

		c := &CloudWatchContext{
			Context:       ctx,
			Event:         event,
			Logger:        logger,
			NewRelicTx:    newrelic.FromContext(ctx),
			CorrelationID: correlationID,
		}

		c.AddNewRelicAttribute("functionName", conf.FunctionName)
		c.AddNewRelicAttribute("buildVersion", conf.BuildVersion)
		c.AddNewRelicAttribute("correlationID", correlationID)
		c.AddNewRelicAttribute("cloudWatchResource", cloudWatchResource)

		result, err := h(c)
		if err != nil {
			logUnhandledError(c.Logger, err)
		}
		return result, err
	}
}

// CloudWatchHandlerWithNewRelic wraps the CloudWatch handler with New Relic monitoring.
func CloudWatchHandlerWithNewRelic(h CloudWatchHandlerFunc, conf HandlerConfig) lambda.Handler {
	return nrlambda.Wrap(CloudWatchHandler(h, conf), conf.NewRelicApp)
}

// AddNewRelicAttribute adds a custom attribute to the New Relic transaction, if available.
func (c *CloudWatchContext) AddNewRelicAttribute(key string, val any) {
	if c.NewRelicTx == nil {
		return
	}
	c.NewRelicTx.AddAttribute(key, val)
}
