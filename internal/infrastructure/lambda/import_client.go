package lambda

import (
	"context"

	awslambda "github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

type ImportClient struct {
	lambdaClient *awslambda.Client
	functionName string
}

func NewImportClient(lambdaClient *awslambda.Client, functionName string) *ImportClient {
	return &ImportClient{lambdaClient: lambdaClient, functionName: functionName}
}

func (c *ImportClient) Invoke(ctx context.Context, payload []byte) error {
	_, err := c.lambdaClient.Invoke(ctx, &awslambda.InvokeInput{
		FunctionName:   &c.functionName,
		InvocationType: types.InvocationTypeEvent,
		Payload:        payload,
	})
	return err
}
