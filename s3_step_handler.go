package g8

import (
	"context"
	newrelic "github.com/newrelic/go-agent"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"

	"github.com/newrelic/go-agent/_integrations/nrlambda"
)

type S3StepHandlerFunc func(c *S3Context) (LambdaEvent, error)

func S3StepHandler(h S3StepHandlerFunc, conf HandlerConfig) func(context.Context, events.S3Event) (LambdaEvent, error) {
	return func(ctx context.Context, e events.S3Event) (LambdaEvent, error) {
		record := e.Records[0]
		correlationID := uuid.New().String()

		logger := configureLogger(conf).
			Str("s3_step_event_source", record.EventSource).
			Str("correlation_id", correlationID).
			Logger()

		c := &S3Context{
			Context:       ctx,
			EventRecord:   record,
			Logger:        logger,
			NewRelicTx:    newrelic.FromContext(ctx),
			CorrelationID: correlationID,
		}

		c.AddNewRelicAttribute("functionName", conf.FunctionName)
		c.AddNewRelicAttribute("buildVersion", conf.BuildVersion)
		c.AddNewRelicAttribute("correlationID", correlationID)
		c.AddNewRelicAttribute("s3StepEventSource", record.EventSource)

		result, err := h(c)

		if err != nil {
			logUnhandledError(c.Logger, err)
		}

		return result, err
	}
}

func S3StepHandlerWithNewRelic(h S3StepHandlerFunc, conf HandlerConfig) lambda.Handler {
	return nrlambda.Wrap(S3StepHandler(h, conf), conf.NewRelicApp)
}
