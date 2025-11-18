package examples

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
	pgModule "github.com/testcontainers/testcontainers-go/modules/postgres"
	redisModule "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	dynamoClient "testcontainers-learning/dynamodb"
	pgClient "testcontainers-learning/postgres"
	redisClient "testcontainers-learning/redis"
)

// TestMultiContainerIntegration은 여러 컨테이너를 동시에 사용하는 통합 테스트입니다
func TestMultiContainerIntegration(t *testing.T) {
	ctx := context.Background()

	// 1. Redis 컨테이너 시작
	redisContainer, err := redisModule.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)
	defer func() {
		if err := testcontainers.TerminateContainer(redisContainer); err != nil {
			t.Logf("failed to terminate redis container: %s", err)
		}
	}()

	redisEndpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	// 2. PostgreSQL 컨테이너 시작
	postgresContainer, err := pgModule.Run(ctx,
		"postgres:16-alpine",
		pgModule.WithDatabase("testdb"),
		pgModule.WithUsername("testuser"),
		pgModule.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	defer func() {
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			t.Logf("failed to terminate postgres container: %s", err)
		}
	}()

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// 3. LocalStack (DynamoDB) 컨테이너 시작
	localstackContainer, err := localstack.Run(ctx, "localstack/localstack:3.0")
	require.NoError(t, err)
	defer func() {
		if err := testcontainers.TerminateContainer(localstackContainer); err != nil {
			t.Logf("failed to terminate localstack container: %s", err)
		}
	}()

	provider, err := testcontainers.NewDockerProvider()
	require.NoError(t, err)
	defer provider.Close()

	host, err := provider.DaemonHost(ctx)
	require.NoError(t, err)

	mappedPort, err := localstackContainer.MappedPort(ctx, "4566/tcp")
	require.NoError(t, err)

	dynamoEndpoint := fmt.Sprintf("http://%s:%s", host, mappedPort.Port())

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	require.NoError(t, err)

	// 4. 클라이언트 생성
	redis := redisClient.NewClient(redisEndpoint)
	defer redis.Close()

	postgres, err := pgClient.NewClient(connStr)
	require.NoError(t, err)
	defer postgres.Close()

	dynamo := dynamoClient.NewClient(cfg, dynamoEndpoint)

	// 5. 통합 테스트 시나리오: 사용자 등록 및 세션 관리

	// PostgreSQL에 사용자 테이블 생성
	err = postgres.CreateTable(ctx, "users")
	require.NoError(t, err)

	// PostgreSQL에 사용자 추가
	userID, err := postgres.InsertUser(ctx, "users", "John Doe", "john@example.com")
	require.NoError(t, err)
	assert.Greater(t, userID, int64(0))

	// Redis에 세션 저장 (캐싱)
	sessionKey := fmt.Sprintf("session:user:%d", userID)
	err = redis.Set(ctx, sessionKey, "active", 1*time.Hour)
	assert.NoError(t, err)

	// DynamoDB에 활동 로그 테이블 생성
	err = dynamo.CreateTable(ctx, "activity_logs")
	require.NoError(t, err)

	// DynamoDB에 사용자 활동 로그 저장
	activity := map[string]types.AttributeValue{
		"id":        &types.AttributeValueMemberS{Value: fmt.Sprintf("log-%d", time.Now().Unix())},
		"user_id":   &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", userID)},
		"action":    &types.AttributeValueMemberS{Value: "login"},
		"timestamp": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", time.Now().Unix())},
	}
	err = dynamo.PutItem(ctx, "activity_logs", activity)
	assert.NoError(t, err)

	// 6. 데이터 검증

	// Redis에서 세션 확인
	session, err := redis.Get(ctx, sessionKey)
	assert.NoError(t, err)
	assert.Equal(t, "active", session)

	// PostgreSQL에서 사용자 조회
	user, err := postgres.GetUser(ctx, "users", userID)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "John Doe", user.Name)
	assert.Equal(t, "john@example.com", user.Email)

	// DynamoDB에서 활동 로그 조회
	logs, err := dynamo.Scan(ctx, "activity_logs")
	assert.NoError(t, err)
	assert.Len(t, logs, 1)

	t.Log("통합 테스트 성공: 모든 컨테이너가 정상적으로 작동하고 데이터가 올바르게 저장되었습니다")
}

// TestCacheAsidePattern은 캐시 어사이드 패턴을 테스트합니다
func TestCacheAsidePattern(t *testing.T) {
	ctx := context.Background()

	// Redis 컨테이너 시작
	redisContainer, err := redisModule.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)
	defer testcontainers.TerminateContainer(redisContainer)

	redisEndpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	// PostgreSQL 컨테이너 시작
	postgresContainer, err := pgModule.Run(ctx,
		"postgres:16-alpine",
		pgModule.WithDatabase("testdb"),
		pgModule.WithUsername("testuser"),
		pgModule.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	defer testcontainers.TerminateContainer(postgresContainer)

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// 클라이언트 생성
	redis := redisClient.NewClient(redisEndpoint)
	defer redis.Close()

	postgres, err := pgClient.NewClient(connStr)
	require.NoError(t, err)
	defer postgres.Close()

	// 테이블 생성 및 데이터 추가
	err = postgres.CreateTable(ctx, "users")
	require.NoError(t, err)

	userID, err := postgres.InsertUser(ctx, "users", "Jane Smith", "jane@example.com")
	require.NoError(t, err)

	// 캐시 어사이드 패턴 구현
	cacheKey := fmt.Sprintf("user:%d", userID)

	// 1. 캐시에서 먼저 조회
	cachedData, err := redis.Get(ctx, cacheKey)
	if err != nil {
		// 2. 캐시 미스 - DB에서 조회
		user, err := postgres.GetUser(ctx, "users", userID)
		require.NoError(t, err)
		require.NotNil(t, user)

		// 3. 캐시에 저장
		cacheValue := fmt.Sprintf("%s:%s", user.Name, user.Email)
		err = redis.Set(ctx, cacheKey, cacheValue, 5*time.Minute)
		require.NoError(t, err)

		t.Log("캐시 미스 - DB에서 조회 후 캐시에 저장")
	} else {
		t.Logf("캐시 히트 - 캐시된 데이터: %s", cachedData)
	}

	// 4. 다시 조회 - 이번에는 캐시 히트
	cachedData, err = redis.Get(ctx, cacheKey)
	assert.NoError(t, err)
	assert.Equal(t, "Jane Smith:jane@example.com", cachedData)

	t.Log("캐시 어사이드 패턴 테스트 성공")
}

// TestDistributedCounter는 분산 카운터 패턴을 테스트합니다
func TestDistributedCounter(t *testing.T) {
	ctx := context.Background()

	// Redis 컨테이너 시작
	redisContainer, err := redisModule.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)
	defer testcontainers.TerminateContainer(redisContainer)

	redisEndpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	// PostgreSQL 컨테이너 시작
	postgresContainer, err := pgModule.Run(ctx,
		"postgres:16-alpine",
		pgModule.WithDatabase("testdb"),
		pgModule.WithUsername("testuser"),
		pgModule.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	defer testcontainers.TerminateContainer(postgresContainer)

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// 클라이언트 생성
	redis := redisClient.NewClient(redisEndpoint)
	defer redis.Close()

	postgres, err := pgClient.NewClient(connStr)
	require.NoError(t, err)
	defer postgres.Close()

	// 페이지 뷰 카운터 시나리오
	pageID := "article-123"
	counterKey := fmt.Sprintf("pageviews:%s", pageID)

	// Redis에서 원자적으로 카운터 증가
	for i := 0; i < 10; i++ {
		_, err := redis.Increment(ctx, counterKey)
		require.NoError(t, err)
	}

	// 카운터 값 확인
	count, err := redis.Get(ctx, counterKey)
	assert.NoError(t, err)
	assert.Equal(t, "10", count)

	// 주기적으로 DB에 동기화하는 시나리오
	// (실제로는 백그라운드 작업으로 처리)
	// 여기서는 테스트를 위해 수동으로 수행

	t.Logf("페이지 %s의 조회수: %s", pageID, count)
	t.Log("분산 카운터 패턴 테스트 성공")
}
