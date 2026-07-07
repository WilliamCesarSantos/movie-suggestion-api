package sqs

import (
	"context"
	"encoding/json"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/http/middleware"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/rs/zerolog/log"
)

type Publisher struct {
	client   *awssqs.Client
	queueURL string
}

func NewPublisher(client *awssqs.Client, queueURL string) *Publisher {
	return &Publisher{client: client, queueURL: queueURL}
}

func (p *Publisher) Publish(ctx context.Context, imdbID string) error {
	logger := log.Ctx(ctx).With().Str("logger", "sqs.publisher").Logger()
	logger.Info().Str("imdbId", imdbID).Msg("publishing import message")

	msg := ImportMessage{ImdbID: imdbID}
	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error().Err(err).Str("imdbId", imdbID).Msg("failed to marshal import message")
		return err
	}
	body := string(data)

	attrs := map[string]types.MessageAttributeValue{}
	if cid, ok := ctx.Value(middleware.ContextKeyCorrelationID).(string); ok && cid != "" {
		dataType := "String"
		attrs["correlationId"] = types.MessageAttributeValue{
			DataType:    &dataType,
			StringValue: &cid,
		}
	}

	_, err = p.client.SendMessage(ctx, &awssqs.SendMessageInput{
		QueueUrl:          &p.queueURL,
		MessageBody:       &body,
		MessageAttributes: attrs,
	})
	if err != nil {
		logger.Error().Err(err).Str("imdbId", imdbID).Msg("failed to send import message")
		return err
	}
	logger.Info().Str("imdbId", imdbID).Msg("import message published")
	return err
}
