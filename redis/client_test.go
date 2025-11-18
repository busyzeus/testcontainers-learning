package redis

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

var (
	testClient   *Client
	testEndpoint string
)

func TestMain(m *testing.M) {

	// 타임아웃 설정
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Redis 컨테이너 시작 (모든 테스트에서 공유)
	redisContainer, err := redis.Run(ctx,
		"redis:7.2",
		redis.WithSnapshotting(10, 1),
		redis.WithLogLevel(redis.LogLevelVerbose),
	)
	if err != nil {
		panic(err)
	}

	// 연결 정보 얻기
	testEndpoint, err = redisContainer.Endpoint(ctx, "")
	if err != nil {
		_ = testcontainers.TerminateContainer(redisContainer)
		panic(err)
	}

	// 클라이언트 생성
	testClient = NewClient(testEndpoint)

	// 연결 확인
	if err := testClient.Ping(ctx); err != nil {
		testClient.Close()
		_ = testcontainers.TerminateContainer(redisContainer)
		panic("failed to ping redis: " + err.Error())
	}

	// 테스트 실행
	code := m.Run()

	// 정리
	if testClient != nil {
		testClient.Close()
	}
	if err := testcontainers.TerminateContainer(redisContainer); err != nil {
		panic(err)
	}

	os.Exit(code)
}

func TestRedisBasicOperations(t *testing.T) {
	ctx := context.Background()
	client := testClient

	// 테스트 전 키 정리
	_ = client.Delete(ctx, "test-key")

	// Ping 테스트
	err := client.Ping(ctx)
	assert.NoError(t, err)

	// Set/Get 테스트
	err = client.Set(ctx, "test-key", "test-value", 0)
	assert.NoError(t, err)

	value, err := client.Get(ctx, "test-key")
	assert.NoError(t, err)
	assert.Equal(t, "test-value", value)

	// Exists 테스트
	count, err := client.Exists(ctx, "test-key")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Delete 테스트
	err = client.Delete(ctx, "test-key")
	assert.NoError(t, err)

	count, err = client.Exists(ctx, "test-key")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestRedisExpiration(t *testing.T) {
	ctx := context.Background()
	client := testClient

	// 테스트 전 키 정리
	_ = client.Delete(ctx, "expiring-key")

	// TTL을 가진 키 설정
	err := client.Set(ctx, "expiring-key", "value", 1*time.Second)
	assert.NoError(t, err)

	// 즉시 조회 - 존재해야 함
	value, err := client.Get(ctx, "expiring-key")
	assert.NoError(t, err)
	assert.Equal(t, "value", value)

	// 2초 대기
	time.Sleep(2 * time.Second)

	// 만료된 키 조회 - 에러 발생
	_, err = client.Get(ctx, "expiring-key")
	assert.Error(t, err)
}

func TestRedisIncrement(t *testing.T) {
	ctx := context.Background()
	client := testClient

	// 테스트 전 키 정리
	_ = client.Delete(ctx, "counter")

	// Increment 테스트
	val, err := client.Increment(ctx, "counter")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), val)

	val, err = client.Increment(ctx, "counter")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), val)

	// Decrement 테스트
	val, err = client.Decrement(ctx, "counter")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), val)
}

func TestRedisHash(t *testing.T) {
	ctx := context.Background()
	client := testClient

	// 테스트 전 키 정리
	_ = client.Delete(ctx, "user:1")

	// HSet 테스트
	err := client.HSet(ctx, "user:1", "name", "John", "email", "john@example.com")
	assert.NoError(t, err)

	// HGet 테스트
	name, err := client.HGet(ctx, "user:1", "name")
	assert.NoError(t, err)
	assert.Equal(t, "John", name)

	email, err := client.HGet(ctx, "user:1", "email")
	assert.NoError(t, err)
	assert.Equal(t, "john@example.com", email)

	// HGetAll 테스트
	all, err := client.HGetAll(ctx, "user:1")
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{
		"name":  "John",
		"email": "john@example.com",
	}, all)
}

func TestRedisList(t *testing.T) {
	ctx := context.Background()
	client := testClient

	// 테스트 전 키 정리
	_ = client.Delete(ctx, "queue", "stack")

	// RPush 테스트
	err := client.RPush(ctx, "queue", "item1", "item2", "item3")
	assert.NoError(t, err)

	// LRange 테스트
	items, err := client.LRange(ctx, "queue", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"item1", "item2", "item3"}, items)

	// LPush 테스트
	err = client.LPush(ctx, "stack", "first", "second", "third")
	assert.NoError(t, err)

	items, err = client.LRange(ctx, "stack", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"third", "second", "first"}, items)
}
