package g8

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/nrlambda"
	"github.com/rs/zerolog"
)

type S3Context struct {
	Context       context.Context
	EventRecord   events.S3EventRecord
	Logger        zerolog.Logger
	NewRelicTx    newrelic.Transaction
	CorrelationID string
}

type S3HandlerFunc func(c *S3Context) error

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

func S3HandlerWithNewRelic(h S3HandlerFunc, conf HandlerConfig) lambda.Handler {
	return nrlambda.Wrap(S3Handler(h, conf), conf.NewRelicApp)
}

func (c *S3Context) AddNewRelicAttribute(key string, val interface{}) {
	if c.NewRelicTx == nil {
		return
	}
	if err := c.NewRelicTx.AddAttribute(key, val); err != nil {
		c.Logger.Error().Msgf("failed to add attr '%s' to new relic tx: %+v", key, err)
	}
}
