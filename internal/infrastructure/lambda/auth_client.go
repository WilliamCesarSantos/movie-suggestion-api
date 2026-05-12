package lambda

import (
	"context"
	"encoding/json"

	awslambda "github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

type AuthResponse struct {
	Valid     bool   `json:"valid"`
	UserID    string `json:"userId"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Error     string `json:"error,omitempty"`
	Token     string `json:"token,omitempty"`
	ExpiresAt string `json:"expiresAt,omitempty"`
}

type AuthClient struct {
	lambdaClient *awslambda.Client
	functionName string
}

func NewAuthClient(lambdaClient *awslambda.Client, functionName string) *AuthClient {
	return &AuthClient{lambdaClient: lambdaClient, functionName: functionName}
}

func (c *AuthClient) Generate(ctx context.Context, userID, email, role string) (*AuthResponse, error) {
	payload := map[string]any{
		"action": "generate",
		"userId": userID,
		"email":  email,
		"role":   role,
	}
	return c.invoke(ctx, payload)
}

func (c *AuthClient) Validate(ctx context.Context, token string) (*AuthResponse, error) {
	payload := map[string]any{
		"action": "validate",
		"token":  token,
	}
	return c.invoke(ctx, payload)
}

func (c *AuthClient) invoke(ctx context.Context, payload map[string]any) (*AuthResponse, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	out, err := c.lambdaClient.Invoke(ctx, &awslambda.InvokeInput{
		FunctionName:   &c.functionName,
		InvocationType: types.InvocationTypeRequestResponse,
		Payload:        data,
	})
	if err != nil {
		return nil, err
	}
	var resp AuthResponse
	if err := json.Unmarshal(out.Payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
