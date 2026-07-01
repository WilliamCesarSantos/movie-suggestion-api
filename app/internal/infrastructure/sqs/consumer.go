package sqs

import (
	"context"
	"encoding/json"

	appusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/application/usecase"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/rs/zerolog"
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

	for {
		select {
		case <-ctx.Done():
			close(msgCh)
			return
		default:
			out, err := c.sqsClient.ReceiveMessage(ctx, &awssqs.ReceiveMessageInput{
				QueueUrl:            &c.queueURL,
				MaxNumberOfMessages: 10,
				WaitTimeSeconds:     20,
			})
			if err != nil {
				c.logger.Error().Str("correlationId", "system").Err(err).Msg("SQS receive error")
				continue
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
		var im ImportMessage
		if err := json.Unmarshal([]byte(*msg.Body), &im); err != nil {
			c.logger.Error().Str("correlationId", "system").Str("messageId", messageID).Err(err).Msg("failed to unmarshal SQS message")
			continue
		}
		if err := c.processor.Process(ctx, im.ImdbID); err != nil {
			c.logger.Error().Str("correlationId", "system").Str("messageId", messageID).Err(err).Str("imdbId", im.ImdbID).Msg("failed to process movie import")
			continue
		}
		_, err := c.sqsClient.DeleteMessage(ctx, &awssqs.DeleteMessageInput{
			QueueUrl:      &c.queueURL,
			ReceiptHandle: msg.ReceiptHandle,
		})
		if err != nil {
			c.logger.Error().Str("correlationId", "system").Str("messageId", messageID).Err(err).Msg("failed to delete SQS message")
		}
	}
}
