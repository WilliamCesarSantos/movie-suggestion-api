package sqs

import (
	"context"
	"encoding/json"

	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
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
	_, err = p.client.SendMessage(ctx, &awssqs.SendMessageInput{
		QueueUrl:    &p.queueURL,
		MessageBody: &body,
	})
	return err
}
