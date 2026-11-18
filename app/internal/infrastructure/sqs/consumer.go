package sqs

import (
	"context"
	"encoding/json"
	"time"

	appusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/application/usecase"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/http/middleware"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const (
	receiveInitialBackoff = 500 * time.Millisecond
	receiveMaxBackoff     = 30 * time.Second
)

type Consumer struct {
	sqsClient   *awssqs.Client
	queueURL    string
	workerCount int
	processor   appusecase.ProcessMovieImportUseCase
	logger      zerolog.Logger
}

func NewConsumer(sqsClient *awssqs.Client, queueURL string, workerCount int, processor appusecase.ProcessMovieImportUseCase, logger zerolog.Logger) *Consumer {
	return &Consumer{
		sqsClient:   sqsClient,
		queueURL:    queueURL,
		workerCount: workerCount,
		processor:   processor,
		logger:      logger,
	}
}

func (c *Consumer) Start(ctx context.Context) {
	msgCh := make(chan types.Message, c.workerCount)

	for i := 0; i < c.workerCount; i++ {
		go c.worker(ctx, msgCh)
	}

	backoff := receiveInitialBackoff
	failures := 0

	for {
		select {
		case <-ctx.Done():
			close(msgCh)
			return
		default:
			out, err := c.sqsClient.ReceiveMessage(ctx, &awssqs.ReceiveMessageInput{
				QueueUrl:              &c.queueURL,
				MaxNumberOfMessages:   10,
				WaitTimeSeconds:       20,
				MessageAttributeNames: []string{"correlationId"},
			})
			if err != nil {
				if ctx.Err() != nil {
					continue
				}
				failures++
				c.logger.Error().
					Str("correlationId", "system").
					Str("queueUrl", c.queueURL).
					Int("consecutiveFailures", failures).
					Dur("retryIn", backoff).
					Err(err).
					Msg("SQS receive error")
				select {
				case <-ctx.Done():
				case <-time.After(backoff):
				}
				backoff = min(backoff*2, receiveMaxBackoff)
				continue
			}
			if failures > 0 {
				c.logger.Info().
					Str("correlationId", "system").
					Str("queueUrl", c.queueURL).
					Int("previousFailures", failures).
					Msg("SQS receive recovered")
				failures = 0
				backoff = receiveInitialBackoff
			}
			for _, msg := range out.Messages {
				select {
				case <-ctx.Done():
					return
				case msgCh <- msg:
				}
			}
		}
	}
}

func (c *Consumer) worker(ctx context.Context, msgCh <-chan types.Message) {
	for msg := range msgCh {
		messageID := ""
		if msg.MessageId != nil {
			messageID = *msg.MessageId
		}

		correlationID := "system"
		if attr, ok := msg.MessageAttributes["correlationId"]; ok && attr.StringValue != nil {
			correlationID = *attr.StringValue
		} else {
			correlationID = uuid.New().String()
		}

		msgCtx := context.WithValue(ctx, middleware.ContextKeyCorrelationID, correlationID)
		msgLogger := c.logger.With().Str("correlationId", correlationID).Str("messageId", messageID).Logger()
		msgCtx = msgLogger.WithContext(msgCtx)

		var im ImportMessage
		if err := json.Unmarshal([]byte(*msg.Body), &im); err != nil {
			msgLogger.Error().Err(err).Msg("failed to unmarshal SQS message")
			continue
		}
		if err := c.processor.Process(msgCtx, im.ImdbID); err != nil {
			msgLogger.Error().Err(err).Str("imdbId", im.ImdbID).Msg("failed to process movie import")
			continue
		}
		_, err := c.sqsClient.DeleteMessage(ctx, &awssqs.DeleteMessageInput{
			QueueUrl:      &c.queueURL,
			ReceiptHandle: msg.ReceiptHandle,
		})
		if err != nil {
			msgLogger.Error().Err(err).Msg("failed to delete SQS message")
		}
	}
}
