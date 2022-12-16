package g8

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/nrlambda"
	"github.com/rs/zerolog"
)

type StepContext struct {
	Context       context.Context
	Event         StepEvent
	Logger        zerolog.Logger
	NewRelicTx    newrelic.Transaction
	CorrelationID string
}

type StepHandlerFunc func(c *StepContext) (StepEvent, error)

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

func StepHandlerWithNewRelic(h StepHandlerFunc, conf HandlerConfig) lambda.Handler {
	return nrlambda.Wrap(StepHandler(h, conf), conf.NewRelicApp)
}

func (c *StepContext) AddNewRelicAttribute(key string, val interface{}) {
	if c.NewRelicTx == nil {
		return
	}
	if err := c.NewRelicTx.AddAttribute(key, val); err != nil {
		c.Logger.Error().Msgf("failed to add attr '%s' to new relic tx: %+v", key, err)
	}
}
