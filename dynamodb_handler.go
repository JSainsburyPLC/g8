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

// DynamoDbContext is the context passed to DynamoDB stream handlers.
type DynamoDbContext struct {
	Context       context.Context
	EventRecord   events.DynamoDBEventRecord
	Logger        zerolog.Logger
	NewRelicTx    *newrelic.Transaction
	CorrelationID string
}

// DynamoHandlerFunc is the user-defined function signature for DynamoDB stream handlers.
type DynamoHandlerFunc func(c *DynamoDbContext) error

// DynamoDbHandler is the main handler for DynamoDB stream events.
func DynamoDbHandler(h DynamoHandlerFunc, conf HandlerConfig) func(context.Context, events.DynamoDBEvent) error {
	return func(ctx context.Context, e events.DynamoDBEvent) error {
		for _, record := range e.Records {
			correlationID := uuid.New().String()

			logger := configureLogger(conf).
				Str("dynamodb_event_source", record.EventSource).
				Str("correlation_id", correlationID).
				Logger()

			c := &DynamoDbContext{
				Context:       ctx,
				EventRecord:   record,
				Logger:        logger,
				NewRelicTx:    newrelic.FromContext(ctx),
				CorrelationID: correlationID,
			}

			c.AddNewRelicAttribute("functionName", conf.FunctionName)
			c.AddNewRelicAttribute("buildVersion", conf.BuildVersion)
			c.AddNewRelicAttribute("correlationID", correlationID)
			c.AddNewRelicAttribute("dynamoDBEventSource", record.EventSource)

			if err := h(c); err != nil {
				logUnhandledError(c.Logger, err)
				return err
			}
		}
		return nil
	}
}

// DynamoDbHandlerWithNewRelic wraps the DynamoDbHandler with New Relic monitoring.
func DynamoDbHandlerWithNewRelic(h DynamoHandlerFunc, conf HandlerConfig) lambda.Handler {
	return nrlambda.Wrap(DynamoDbHandler(h, conf), conf.NewRelicApp)
}

// AddNewRelicAttribute adds a custom attribute to the New Relic transaction, if available.
func (c *DynamoDbContext) AddNewRelicAttribute(key string, val any) {
	if c.NewRelicTx == nil {
		return
	}
	c.NewRelicTx.AddAttribute(key, val)
}
