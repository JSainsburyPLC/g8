package g8_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/JSainsburyPLC/g8"
)

func TestDynamoDbHandler_SingleMessage(t *testing.T) {
	timesCalled := 0
	h := g8.DynamoDbHandler(func(c *g8.DynamoDbContext) error {
		timesCalled++

		assert.Equal(t, "event1", c.EventRecord.EventName)
		assert.IsType(t, events.DynamoDBStreamRecord{}, c.EventRecord.Change)
		assert.NotEmpty(t, c.CorrelationID)

		return nil
	}, g8.HandlerConfig{Logger: zerolog.New(ioutil.Discard)})

	err := h(context.Background(), events.DynamoDBEvent{
		Records: []events.DynamoDBEventRecord{
			{
				EventName: "event1",
				Change:    events.DynamoDBStreamRecord{},
			},
		},
	})

	assert.Nil(t, err)
	assert.Equal(t, 1, timesCalled)
}

func TestDynamoDbHandler_MultipleMessages(t *testing.T) {
	timesCalled := 0
	h := g8.DynamoDbHandler(func(c *g8.DynamoDbContext) error {
		timesCalled++

		assert.Equal(t, fmt.Sprintf("event-%d", timesCalled), c.EventRecord.EventName)
		assert.NotEmpty(t, c.CorrelationID)
		assert.IsType(t, events.DynamoDBStreamRecord{}, c.EventRecord.Change)

		return nil
	}, g8.HandlerConfig{Logger: zerolog.New(ioutil.Discard)})

	err := h(context.Background(), events.DynamoDBEvent{
		Records: []events.DynamoDBEventRecord{
			{
				EventName: "event-1",
				Change:    events.DynamoDBStreamRecord{},
			},
			{
				EventName: "event-2",
				Change:    events.DynamoDBStreamRecord{},
			},
		},
	})

	assert.Nil(t, err)
	assert.Equal(t, 2, timesCalled)
}

func TestDynamoDbHandler_HandlerError(t *testing.T) {
	timesCalled := 0
	handlerFunc := func(c *g8.DynamoDbContext) error {
		timesCalled++
		return assert.AnError
	}

	h := g8.DynamoDbHandler(handlerFunc, g8.HandlerConfig{
		Logger: zerolog.New(ioutil.Discard),
	})
	err := h(context.Background(), events.DynamoDBEvent{
		Records: []events.DynamoDBEventRecord{
			{
				EventName: "event-1",
				Change:    events.DynamoDBStreamRecord{},
			},
		},
	})

	assert.Equal(t, assert.AnError, err)
	assert.Equal(t, 1, timesCalled)
}
