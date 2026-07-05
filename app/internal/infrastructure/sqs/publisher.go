package sqs

import (
	"context"
	"encoding/json"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/http/middleware"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type Publisher struct {
	client   *awssqs.Client
	queueURL string
}

func NewPublisher(client *awssqs.Client, queueURL string) *Publisher {
	return &Publisher{client: client, queueURL: queueURL}
}

func (p *Publisher) Publish(ctx context.Context, imdbID string) error {
	msg := ImportMessage{ImdbID: imdbID}
	data, err := json.Marshal(msg)
	if err != nil {
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
	return err
}
