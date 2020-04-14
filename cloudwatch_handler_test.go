package g8_test

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/JSainsburyPLC/g8"
)

func TestCloudWatchHandler_SingleMessage(t *testing.T) {
	timesCalled := 0
	resourceArn := "arn:aws:events:us-east-1:123456789012:rule/MyScheduledRule"
	h := g8.CloudWatchHandler(func(c *g8.CloudWatchContext) (g8.LambdaResult, error) {
		timesCalled++

		assert.Equal(t, resourceArn, c.Event.Resources[0])
		assert.NotEmpty(t, c.CorrelationID)

		return "finished", nil
	}, g8.HandlerConfig{Logger: zerolog.New(ioutil.Discard)})

	result, err := h(context.Background(), events.CloudWatchEvent{
		Resources: []string{resourceArn},
	})

	assert.Nil(t, err)
	assert.Equal(t, "finished", result)
	assert.Equal(t, 1, timesCalled)
}

func TestCloudWatchHandler_HandlerError(t *testing.T) {
	timesCalled := 0
	handlerFunc := func(c *g8.CloudWatchContext) (g8.LambdaResult, error) {
		timesCalled++
		return nil, assert.AnError
	}

	h := g8.CloudWatchHandler(handlerFunc, g8.HandlerConfig{
		Logger: zerolog.New(ioutil.Discard),
	})
	_, err := h(context.Background(), events.CloudWatchEvent{})

	assert.Equal(t, assert.AnError, err)
	assert.Equal(t, 1, timesCalled)
}
