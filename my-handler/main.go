package main

import (
	"context"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"golang.org/x/net/context/ctxhttp"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/kelseyhightower/envconfig"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"
)

type handlerConfig struct {
	DynamodbTableName string `envconfig:"DYNAMODB_TABLE_NAME" required:"true"`
}

type lambdaHandler struct {
	dynamodbTableName string
}

type dynamodbItem struct {
	ID   string `dynamodbav:"id,omitempty"`
	Text string `dynamodbav:"text,omitempty"`
	TTL  int64  `dynamodbav:"ttl,omitempty"`
}

func (lh *lambdaHandler) handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	err := xray.Configure(xray.Config{LogLevel: "warn"})
	if err != nil {
		log.WithError(err).Error("error initializing X-Ray")
		return events.APIGatewayProxyResponse{Body: "{}", StatusCode: 500}, nil
	}

	sess := session.Must(session.NewSession())
	dynamodbClient := dynamodb.New(sess)
	xray.AWS(dynamodbClient.Client)

	attributeValues, err := dynamodbattribute.MarshalMap(&dynamodbItem{
		ID:   uuid.New(),
		Text: "test",
		TTL:  time.Now().UTC().Add(6 * time.Hour).Unix(),
	})
	if err != nil {
		log.WithError(err).Error("failed to marshal DynamoDB item")
		return events.APIGatewayProxyResponse{Body: "{}", StatusCode: 500}, nil
	}

	_, err = dynamodbClient.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(lh.dynamodbTableName),
		Item:      attributeValues,
	})
	if err != nil {
		log.WithError(err).WithField("dynamodb_table_name", lh.dynamodbTableName).Error("could not write to DynamoDB table")
		return events.APIGatewayProxyResponse{Body: "{}", StatusCode: 500}, nil
	}

	httpClient := xray.Client(http.DefaultClient)
	url := "http://example.com/"
	_, err = ctxhttp.Get(ctx, httpClient, url)
	if err != nil {
		log.WithError(err).WithField("url", url).Error("could not get data from URL")
		return events.APIGatewayProxyResponse{Body: "{}", StatusCode: 500}, nil
	}

	return events.APIGatewayProxyResponse{Body: "{}", StatusCode: 200}, nil
}

func main() {
	var c handlerConfig
	err := envconfig.Process("", &c)
	if err != nil {
		log.WithError(err).Fatal("Error parsing configuration")
	}

	lh := &lambdaHandler{
		dynamodbTableName: c.DynamodbTableName,
	}
	lambda.Start(lh.handler)
}
