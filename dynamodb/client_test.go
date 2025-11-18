package dynamodb

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
)

var (
	testClient *Client
)

func TestMain(m *testing.M) {
	// RYUK 비활성화
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// LocalStack 컨테이너 시작 (DynamoDB 포함)
	localstackContainer, err := localstack.Run(ctx,
		"localstack/localstack:3.0",
	)
	if err != nil {
		panic(err)
	}

	// Provider를 통해 호스트 정보 얻기
	provider, err := testcontainers.NewDockerProvider()
	if err != nil {
		panic(err)
	}

	host, err := provider.DaemonHost(ctx)
	if err != nil {
		provider.Close()
		_ = testcontainers.TerminateContainer(localstackContainer)
		panic(err)
	}

	mappedPort, err := localstackContainer.MappedPort(ctx, "4566/tcp")
	if err != nil {
		provider.Close()
		_ = testcontainers.TerminateContainer(localstackContainer)
		panic(err)
	}

	endpoint := fmt.Sprintf("http://%s:%s", host, mappedPort.Port())

	// AWS 설정
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	if err != nil {
		provider.Close()
		_ = testcontainers.TerminateContainer(localstackContainer)
		panic(err)
	}

	testClient = NewClient(cfg, endpoint)

	// 테스트 실행
	code := m.Run()

	// 정리
	provider.Close()
	if err := testcontainers.TerminateContainer(localstackContainer); err != nil {
		panic(err)
	}

	os.Exit(code)
}

func TestDynamoDBCreateTable(t *testing.T) {
	client := testClient

	ctx := context.Background()
	tableName := "test-table"

	// 테스트 전 테이블 정리
	_ = client.DeleteTable(ctx, tableName)

	// 테이블 생성
	err := client.CreateTable(ctx, tableName)
	assert.NoError(t, err)

	// 테이블 정보 조회
	output, err := client.DescribeTable(ctx, tableName)
	assert.NoError(t, err)
	assert.Equal(t, tableName, *output.Table.TableName)
	assert.Equal(t, types.TableStatusActive, output.Table.TableStatus)

	// 테스트 후 정리
	_ = client.DeleteTable(ctx, tableName)
}

func TestDynamoDBPutAndGetItem(t *testing.T) {
	client := testClient
	ctx := context.Background()
	tableName := "users"

	// 테스트 전 테이블 정리
	_ = client.DeleteTable(ctx, tableName)

	// 테이블 생성
	err := client.CreateTable(ctx, tableName)
	require.NoError(t, err)

	// 항목 추가
	item := map[string]types.AttributeValue{
		"id":    &types.AttributeValueMemberS{Value: "user-1"},
		"name":  &types.AttributeValueMemberS{Value: "John Doe"},
		"email": &types.AttributeValueMemberS{Value: "john@example.com"},
		"age":   &types.AttributeValueMemberN{Value: "30"},
	}

	err = client.PutItem(ctx, tableName, item)
	assert.NoError(t, err)

	// 항목 조회
	key := map[string]types.AttributeValue{
		"id": &types.AttributeValueMemberS{Value: "user-1"},
	}

	result, err := client.GetItem(ctx, tableName, key)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// 결과 검증
	nameAttr, ok := result["name"].(*types.AttributeValueMemberS)
	assert.True(t, ok)
	assert.Equal(t, "John Doe", nameAttr.Value)

	emailAttr, ok := result["email"].(*types.AttributeValueMemberS)
	assert.True(t, ok)
	assert.Equal(t, "john@example.com", emailAttr.Value)
}

func TestDynamoDBUpdateItem(t *testing.T) {
	client := testClient
	ctx := context.Background()
	tableName := "users-update"

	// 테스트 전 테이블 정리
	_ = client.DeleteTable(ctx, tableName)

	// 테이블 생성
	err := client.CreateTable(ctx, tableName)
	require.NoError(t, err)

	// 초기 항목 추가
	item := map[string]types.AttributeValue{
		"id":   &types.AttributeValueMemberS{Value: "user-1"},
		"name": &types.AttributeValueMemberS{Value: "John Doe"},
	}
	err = client.PutItem(ctx, tableName, item)
	require.NoError(t, err)

	// 항목 업데이트
	key := map[string]types.AttributeValue{
		"id": &types.AttributeValueMemberS{Value: "user-1"},
	}
	updateExpression := "SET #n = :name"
	expressionAttributeNames := map[string]string{
		"#n": "name",
	}
	expressionAttributeValues := map[string]types.AttributeValue{
		":name": &types.AttributeValueMemberS{Value: "Jane Doe"},
	}

	err = client.UpdateItem(ctx, tableName, key, updateExpression, expressionAttributeValues, expressionAttributeNames)
	assert.NoError(t, err)

	// 업데이트된 항목 조회
	result, err := client.GetItem(ctx, tableName, key)
	assert.NoError(t, err)

	nameAttr, ok := result["name"].(*types.AttributeValueMemberS)
	assert.True(t, ok)
	assert.Equal(t, "Jane Doe", nameAttr.Value)
}

func TestDynamoDBDeleteItem(t *testing.T) {
	client := testClient
	ctx := context.Background()
	tableName := "users-delete"

	// 테스트 전 테이블 정리
	_ = client.DeleteTable(ctx, tableName)

	// 테이블 생성
	err := client.CreateTable(ctx, tableName)
	require.NoError(t, err)

	// 항목 추가
	item := map[string]types.AttributeValue{
		"id":   &types.AttributeValueMemberS{Value: "user-1"},
		"name": &types.AttributeValueMemberS{Value: "John Doe"},
	}
	err = client.PutItem(ctx, tableName, item)
	require.NoError(t, err)

	// 항목 삭제
	key := map[string]types.AttributeValue{
		"id": &types.AttributeValueMemberS{Value: "user-1"},
	}
	err = client.DeleteItem(ctx, tableName, key)
	assert.NoError(t, err)

	// 삭제된 항목 조회 시도
	result, err := client.GetItem(ctx, tableName, key)
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestDynamoDBScan(t *testing.T) {
	client := testClient
	ctx := context.Background()
	tableName := "users-scan"

	// 테스트 전 테이블 정리
	_ = client.DeleteTable(ctx, tableName)

	// 테이블 생성
	err := client.CreateTable(ctx, tableName)
	require.NoError(t, err)

	// 여러 항목 추가
	items := []map[string]types.AttributeValue{
		{
			"id":   &types.AttributeValueMemberS{Value: "user-1"},
			"name": &types.AttributeValueMemberS{Value: "John Doe"},
		},
		{
			"id":   &types.AttributeValueMemberS{Value: "user-2"},
			"name": &types.AttributeValueMemberS{Value: "Jane Smith"},
		},
		{
			"id":   &types.AttributeValueMemberS{Value: "user-3"},
			"name": &types.AttributeValueMemberS{Value: "Bob Johnson"},
		},
	}

	for _, item := range items {
		err = client.PutItem(ctx, tableName, item)
		require.NoError(t, err)
	}

	// 전체 스캔
	results, err := client.Scan(ctx, tableName)
	assert.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestDynamoDBQuery(t *testing.T) {
	client := testClient
	ctx := context.Background()
	tableName := "users-query"

	// 테스트 전 테이블 정리
	_ = client.DeleteTable(ctx, tableName)

	// 테이블 생성
	err := client.CreateTable(ctx, tableName)
	require.NoError(t, err)

	// 항목 추가
	item := map[string]types.AttributeValue{
		"id":   &types.AttributeValueMemberS{Value: "user-1"},
		"name": &types.AttributeValueMemberS{Value: "John Doe"},
	}
	err = client.PutItem(ctx, tableName, item)
	require.NoError(t, err)

	// 쿼리
	keyConditionExpression := "id = :id"
	expressionAttributeValues := map[string]types.AttributeValue{
		":id": &types.AttributeValueMemberS{Value: "user-1"},
	}

	results, err := client.Query(ctx, tableName, keyConditionExpression, expressionAttributeValues)
	assert.NoError(t, err)
	assert.Len(t, results, 1)

	nameAttr, ok := results[0]["name"].(*types.AttributeValueMemberS)
	assert.True(t, ok)
	assert.Equal(t, "John Doe", nameAttr.Value)
}
