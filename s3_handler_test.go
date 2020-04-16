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

func TestS3Handler_SingleMessage(t *testing.T) {
	timesCalled := 0
	h := g8.S3Handler(func(c *g8.S3Context) error {
		timesCalled++

		assert.Equal(t, "12345", c.EventRecord.S3.Object.Key)
		assert.NotEmpty(t, c.CorrelationID)

		return nil
	}, g8.HandlerConfig{Logger: zerolog.New(ioutil.Discard)})

	err := h(context.Background(), events.S3Event{
		Records: []events.S3EventRecord{
			{
				S3: events.S3Entity{
					Object: events.S3Object{
						Key: "12345",
					},
				},
			},
		}})

	assert.Nil(t, err)
	assert.Equal(t, 1, timesCalled)
}

func TestS3Handler_MultipleMessages(t *testing.T) {
	timesCalled := 0
	h := g8.S3Handler(func(c *g8.S3Context) error {
		timesCalled++

		assert.Equal(t, fmt.Sprintf("key-%d", timesCalled), c.EventRecord.S3.Object.Key)
		assert.NotEmpty(t, c.CorrelationID)

		return nil
	}, g8.HandlerConfig{Logger: zerolog.New(ioutil.Discard)})

	err := h(context.Background(), events.S3Event{
		Records: []events.S3EventRecord{
			{
				S3: events.S3Entity{
					Object: events.S3Object{
						Key: "key-1",
					},
				},
			},
			{
				S3: events.S3Entity{
					Object: events.S3Object{
						Key: "key-2",
					},
				},
			},
		}})

	assert.Nil(t, err)
	assert.Equal(t, 2, timesCalled)
}

func TestS3Handler_HandlerError(t *testing.T) {
	timesCalled := 0
	handlerFunc := func(c *g8.S3Context) error {
		timesCalled++
		return assert.AnError
	}

	h := g8.S3Handler(handlerFunc, g8.HandlerConfig{
		Logger: zerolog.New(ioutil.Discard),
	})
	err := h(context.Background(), events.S3Event{
		Records: []events.S3EventRecord{
			{
				S3: events.S3Entity{
					Object: events.S3Object{
						Key: "key-2",
					},
				},
			},
		}})

	assert.Equal(t, assert.AnError, err)
	assert.Equal(t, 1, timesCalled)
}
