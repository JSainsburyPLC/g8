package g8

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/nrlambda"
	"github.com/rs/zerolog"
)

type SQSContext struct {
	Context       context.Context
	Message       events.SQSMessage
	Logger        zerolog.Logger
	NewRelicTx    newrelic.Transaction
	CorrelationID string
}

type SQSHandlerFunc func(c *SQSContext) error

type SQSMessageEnvelope struct {
	Data interface{}     `json:"data"`
	Meta *SQSMessageMeta `json:"meta"`
}

type SQSMessageMeta struct {
	CorrelationID string `json:"correlation_id"`
}

func SQSHandler(h SQSHandlerFunc, conf HandlerConfig) func(context.Context, events.SQSEvent) error {
	return func(ctx context.Context, e events.SQSEvent) error {
		for _, record := range e.Records {
			// parse the envelope and get the meta data if available
			// the body should then be updated with the inner message
			// data for when the data is bound.
			meta, dataBytes := parseRawMessage([]byte(record.Body))
			record.Body = string(dataBytes)

			correlationID := getCorrelationIDSQS(meta)

			logger := conf.Logger.With().
				Str("application", conf.AppName).
				Str("function_name", conf.FunctionName).
				Str("env", conf.EnvName).
				Str("build_version", conf.BuildVersion).
				Str("correlation_id", correlationID).
				Str("sqs_event_source", record.EventSource).
				Str("sqs_message_id", record.MessageId).
				Logger()

			c := &SQSContext{
				Context:       ctx,
				Message:       record,
				Logger:        logger,
				NewRelicTx:    newrelic.FromContext(ctx),
				CorrelationID: correlationID,
			}

			c.AddNewRelicAttribute("functionName", conf.FunctionName)
			c.AddNewRelicAttribute("sqsEventSource", record.EventSource)
			c.AddNewRelicAttribute("sqsMessageID", record.MessageId)
			c.AddNewRelicAttribute("correlationID", correlationID)
			c.AddNewRelicAttribute("buildVersion", conf.BuildVersion)

			if err := h(c); err != nil {
				logUnhandledError(c.Logger, err)
				return err
			}
		}
		return nil
	}
}

func SQSHandlerWithNewRelic(h SQSHandlerFunc, conf HandlerConfig) lambda.Handler {
	return nrlambda.Wrap(SQSHandler(h, conf), conf.NewRelicApp)
}

func (c *SQSContext) AddNewRelicAttribute(key string, val interface{}) {
	if c.NewRelicTx == nil {
		return
	}
	if err := c.NewRelicTx.AddAttribute(key, val); err != nil {
		c.Logger.Error().Msgf("failed to add attr '%s' to new relic tx: %+v", key, err)
	}
}

func (c *SQSContext) Bind(v interface{}) error {
	if err := json.Unmarshal([]byte(c.Message.Body), v); err != nil {
		return err
	}

	if validatable, ok := v.(Validatable); ok {
		err := validatable.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

func parseRawMessage(body []byte) (*SQSMessageMeta, []byte) {
	var envelope SQSMessageEnvelope
	err := json.Unmarshal(body, &envelope)
	if err != nil {
		// errors are swallowed here because the message body isn't guaranteed to be
		// in the envelope type - if not, we should return the entire body
		return nil, body
	}

	if envelope.Meta == nil {
		// if meta is nil, it means the data wasn't enveloped
		return nil, body
	}

	// we want to return the raw JSON of the inner message so it can be bound by the application
	b, err := json.Marshal(envelope.Data)
	if err != nil {
		return nil, body
	}

	return envelope.Meta, b
}

func getCorrelationIDSQS(m *SQSMessageMeta) string {
	if m == nil || m.CorrelationID == "" {
		return uuid.New().String()
	}
	return m.CorrelationID
}
