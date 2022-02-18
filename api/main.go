package main

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type Response events.APIGatewayProxyResponse

func main() {
	lambda.Start(Handler)
}

// Handler is a handler for the API Gateway endpoint
func Handler(_ context.Context) (Response, error) {
	var buf bytes.Buffer

	body, err := json.Marshal(map[string]interface{}{
		"message": "Function executed WAFully!",
	})
	if err != nil {
		return Response{StatusCode: 404}, err
	}
	json.HTMLEscape(&buf, body)

	return Response{
		StatusCode: 200,
		Body:       buf.String(),
		Headers:    map[string]string{"Content-Type": "application/json"},
	}, nil
}
