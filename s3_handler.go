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

// S3Context is the context passed to S3 event handlers.
type S3Context struct {
	Context       context.Context
	EventRecord   events.S3EventRecord
	Logger        zerolog.Logger
	NewRelicTx    *newrelic.Transaction
	CorrelationID string
}

// S3HandlerFunc is the user-defined function signature for S3 event handlers.
type S3HandlerFunc func(c *S3Context) error

// S3Handler is the main handler for S3 events.
func S3Handler(h S3HandlerFunc, conf HandlerConfig) func(context.Context, events.S3Event) error {
	return func(ctx context.Context, e events.S3Event) error {
		for _, record := range e.Records {
			correlationID := uuid.New().String()

			logger := configureLogger(conf).
				Str("s3_event_source", record.EventSource).
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
			c.AddNewRelicAttribute("s3EventSource", record.EventSource)

			if err := h(c); err != nil {
				logUnhandledError(c.Logger, err)
				return err
			}
		}
		return nil
	}
}

// S3HandlerWithNewRelic wraps the S3Handler with New Relic monitoring.
func S3HandlerWithNewRelic(h S3HandlerFunc, conf HandlerConfig) lambda.Handler {
	return nrlambda.Wrap(S3Handler(h, conf), conf.NewRelicApp)
}

// AddNewRelicAttribute adds a custom attribute to the New Relic transaction, if available.
func (c *S3Context) AddNewRelicAttribute(key string, val any) {
	if c.NewRelicTx == nil {
		return
	}
	c.NewRelicTx.AddAttribute(key, val)
}
