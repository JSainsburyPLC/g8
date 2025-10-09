package g8

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	"github.com/newrelic/go-agent/v3/integrations/nrlambda"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog"
)

// StepEvent represents a generic event for AWS Step Functions.
type StepContext struct {
	Context       context.Context
	Event         StepEvent
	Logger        zerolog.Logger
	NewRelicTx    *newrelic.Transaction
	CorrelationID string
}

// StepEvent represents a generic event for AWS Step Functions.
type StepHandlerFunc func(c *StepContext) (StepEvent, error)

// StepHandler is the main handler for Step Function events.
func StepHandler(h StepHandlerFunc, conf HandlerConfig) func(context.Context, StepEvent) (StepEvent, error) {
	return func(ctx context.Context, e StepEvent) (StepEvent, error) {
		correlationID := uuid.New().String()

		logger := configureLogger(conf).
			Str("event_source", "step_function_event").
			Str("correlation_id", correlationID).
			Logger()

		c := &StepContext{
			Context:       ctx,
			Event:         e,
			Logger:        logger,
			NewRelicTx:    newrelic.FromContext(ctx),
			CorrelationID: correlationID,
		}

		c.AddNewRelicAttribute("functionName", conf.FunctionName)
		c.AddNewRelicAttribute("buildVersion", conf.BuildVersion)
		c.AddNewRelicAttribute("correlationID", correlationID)
		c.AddNewRelicAttribute("eventSource", "step_function_event")

		result, err := h(c)

		if err != nil {
			logUnhandledError(c.Logger, err)
		}

		return result, err
	}
}

// StepHandlerWithNewRelic wraps the StepHandler with New Relic monitoring.
func StepHandlerWithNewRelic(h StepHandlerFunc, conf HandlerConfig) lambda.Handler {
	return nrlambda.Wrap(StepHandler(h, conf), conf.NewRelicApp)
}

// AddNewRelicAttribute adds a custom attribute to the New Relic transaction if it exists.
func (c *StepContext) AddNewRelicAttribute(key string, val any) {
	if c.NewRelicTx == nil {
		return
	}
	c.NewRelicTx.AddAttribute(key, val)
}
