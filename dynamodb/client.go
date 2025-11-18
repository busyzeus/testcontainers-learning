package dynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Client는 DynamoDB 클라이언트를 래핑합니다
type Client struct {
	ddb *dynamodb.Client
}

// NewClient는 새로운 DynamoDB 클라이언트를 생성합니다
func NewClient(cfg aws.Config, endpoint string) *Client {
	return &Client{
		ddb: dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		}),
	}
}

// CreateTable은 새로운 테이블을 생성합니다
func (c *Client) CreateTable(ctx context.Context, tableName string) error {
	_, err := c.ddb.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       types.KeyTypeHash,
			},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	return err
}

// DescribeTable은 테이블 정보를 조회합니다
func (c *Client) DescribeTable(ctx context.Context, tableName string) (*dynamodb.DescribeTableOutput, error) {
	return c.ddb.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
}

// DeleteTable은 테이블을 삭제합니다
func (c *Client) DeleteTable(ctx context.Context, tableName string) error {
	_, err := c.ddb.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	})
	return err
}

// PutItem은 항목을 추가합니다
func (c *Client) PutItem(ctx context.Context, tableName string, item map[string]types.AttributeValue) error {
	_, err := c.ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	})
	return err
}

// GetItem은 항목을 조회합니다
func (c *Client) GetItem(ctx context.Context, tableName string, key map[string]types.AttributeValue) (map[string]types.AttributeValue, error) {
	result, err := c.ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key:       key,
	})
	if err != nil {
		return nil, err
	}
	return result.Item, nil
}

// UpdateItem은 항목을 업데이트합니다
func (c *Client) UpdateItem(ctx context.Context, tableName string, key map[string]types.AttributeValue, updateExpression string, expressionAttributeValues map[string]types.AttributeValue) error {
	_, err := c.ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(tableName),
		Key:                       key,
		UpdateExpression:          aws.String(updateExpression),
		ExpressionAttributeValues: expressionAttributeValues,
	})
	return err
}

// DeleteItem은 항목을 삭제합니다
func (c *Client) DeleteItem(ctx context.Context, tableName string, key map[string]types.AttributeValue) error {
	_, err := c.ddb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key:       key,
	})
	return err
}

// Query는 조건 기반 쿼리를 수행합니다
func (c *Client) Query(ctx context.Context, tableName string, keyConditionExpression string, expressionAttributeValues map[string]types.AttributeValue) ([]map[string]types.AttributeValue, error) {
	result, err := c.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:                 aws.String(tableName),
		KeyConditionExpression:    aws.String(keyConditionExpression),
		ExpressionAttributeValues: expressionAttributeValues,
	})
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// Scan은 전체 테이블을 스캔합니다
func (c *Client) Scan(ctx context.Context, tableName string) ([]map[string]types.AttributeValue, error) {
	result, err := c.ddb.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}
