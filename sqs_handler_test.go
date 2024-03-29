package g8_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/JSainsburyPLC/g8"
)

func TestSQSHandler_SingleMessage(t *testing.T) {
	timesCalled := 0
	handlerFunc := func(c *g8.SQSContext) error {
		timesCalled++
		var data map[string]string
		err := c.Bind(&data)
		if err != nil {
			return err
		}

		assert.Equal(t, "value1", data["key1"])
		assert.Equal(t, "abcdef", c.CorrelationID)

		return nil
	}

	h := g8.SQSHandler(handlerFunc, g8.HandlerConfig{
		Logger: zerolog.New(io.Discard),
	})
	err := h(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{
			Body: `{
                     "data": {
                       "key1": "value1",
                       "key2": "value2"
                     },
                     "meta": {
                       "correlation_id": "abcdef"
                     }
                   }`,
		},
	}})

	assert.Nil(t, err)
	assert.Equal(t, 1, timesCalled)
}

func TestSQSHandler_MultipleMessages(t *testing.T) {
	timesCalled := 0
	handlerFunc := func(c *g8.SQSContext) error {
		timesCalled++
		var data map[string]string
		err := c.Bind(&data)
		if err != nil {
			return err
		}

		assert.Equal(t, fmt.Sprintf("message-%d", timesCalled), data["message"])
		assert.Equal(t, "abcdef", c.CorrelationID)

		return nil
	}

	h := g8.SQSHandler(handlerFunc, g8.HandlerConfig{
		Logger: zerolog.New(io.Discard),
	})
	err := h(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{
			Body: `{
                     "data": {
                       "message": "message-1"
                     },
                     "meta": {
                       "correlation_id": "abcdef"
                     }
                   }`,
		},
		{
			Body: `{
                     "data": {
                       "message": "message-2"
                     },
                     "meta": {
                       "correlation_id": "abcdef"
                     }
                   }`,
		},
	}})

	assert.Nil(t, err)
	assert.Equal(t, 2, timesCalled)
}

func TestSQSHandler_EnvelopeNoCorrelationID(t *testing.T) {
	timesCalled := 0
	handlerFunc := func(c *g8.SQSContext) error {
		timesCalled++
		var data map[string]string
		err := c.Bind(&data)
		if err != nil {
			return err
		}

		assert.Equal(t, "value1", data["key1"])
		assert.Len(t, c.CorrelationID, 36)

		return nil
	}

	h := g8.SQSHandler(handlerFunc, g8.HandlerConfig{
		Logger: zerolog.New(io.Discard),
	})
	err := h(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{
			Body: `{
                     "data": {
                       "key1": "value1",
                       "key2": "value2"
                     },
                     "meta": {}
                   }`,
		},
	}})

	assert.Nil(t, err)
	assert.Equal(t, 1, timesCalled)
}

func TestSQSHandler_NoEnvelope(t *testing.T) {
	timesCalled := 0
	handlerFunc := func(c *g8.SQSContext) error {
		timesCalled++
		var data map[string]string
		err := c.Bind(&data)
		if err != nil {
			return err
		}

		assert.Equal(t, "value1", data["key1"])
		assert.Len(t, c.CorrelationID, 36)

		return nil
	}

	h := g8.SQSHandler(handlerFunc, g8.HandlerConfig{
		Logger: zerolog.New(io.Discard),
	})
	err := h(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{
			Body: `{
                     "key1": "value1",
                     "key2": "value2"
                   }`,
		},
	}})

	assert.Nil(t, err)
	assert.Equal(t, 1, timesCalled)
}

func TestSQSHandler_InvalidJSON(t *testing.T) {
	timesCalled := 0
	handlerFunc := func(c *g8.SQSContext) error {
		timesCalled++
		var data map[string]string
		err := c.Bind(&data)
		assert.NotNil(t, err)
		return err
	}

	h := g8.SQSHandler(handlerFunc, g8.HandlerConfig{
		Logger: zerolog.New(io.Discard),
	})
	err := h(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{
			Body: `not valid json`,
		},
	}})

	assert.NotNil(t, err)
	assert.Equal(t, "invalid character 'o' in literal null (expecting 'u')", err.Error())
	assert.Equal(t, 1, timesCalled)
}

func TestSQSHandler_HandlerError(t *testing.T) {
	timesCalled := 0
	handlerFunc := func(c *g8.SQSContext) error {
		timesCalled++
		return assert.AnError
	}

	h := g8.SQSHandler(handlerFunc, g8.HandlerConfig{
		Logger: zerolog.New(io.Discard),
	})
	err := h(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{
			Body: `{
                     "key1": "value1",
                     "key2": "value2"
                   }`,
		},
	}})

	assert.Equal(t, assert.AnError, err)
	assert.Equal(t, 1, timesCalled)
}
