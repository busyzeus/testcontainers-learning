package redis

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestRedisBasicOperations(t *testing.T) {
	ctx := context.Background()

	// Redis 컨테이너 시작
	redisContainer, err := redis.Run(ctx,
		"redis:7-alpine",
	)
	require.NoError(t, err)
	defer func() {
		if err := testcontainers.TerminateContainer(redisContainer); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	}()

	// 연결 정보 얻기
	endpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	// 클라이언트 생성
	client := NewClient(endpoint)
	defer client.Close()

	// Ping 테스트
	err = client.Ping(ctx)
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

	redisContainer, err := redis.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)
	defer func() {
		if err := testcontainers.TerminateContainer(redisContainer); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	}()

	endpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	client := NewClient(endpoint)
	defer client.Close()

	// TTL을 가진 키 설정
	err = client.Set(ctx, "expiring-key", "value", 1*time.Second)
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

	redisContainer, err := redis.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)
	defer func() {
		if err := testcontainers.TerminateContainer(redisContainer); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	}()

	endpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	client := NewClient(endpoint)
	defer client.Close()

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

	redisContainer, err := redis.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)
	defer func() {
		if err := testcontainers.TerminateContainer(redisContainer); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	}()

	endpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	client := NewClient(endpoint)
	defer client.Close()

	// HSet 테스트
	err = client.HSet(ctx, "user:1", "name", "John", "email", "john@example.com")
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

	redisContainer, err := redis.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)
	defer func() {
		if err := testcontainers.TerminateContainer(redisContainer); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	}()

	endpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	client := NewClient(endpoint)
	defer client.Close()

	// RPush 테스트
	err = client.RPush(ctx, "queue", "item1", "item2", "item3")
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
