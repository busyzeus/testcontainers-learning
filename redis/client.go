package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client는 Redis 클라이언트를 래핑합니다
type Client struct {
	rdb *redis.Client
}

// NewClient는 새로운 Redis 클라이언트를 생성합니다
func NewClient(addr string) *Client {
	return &Client{
		rdb: redis.NewClient(&redis.Options{
			Addr: addr,
		}),
	}
}

// Close는 Redis 연결을 종료합니다
func (c *Client) Close() error {
	return c.rdb.Close()
}

// Ping은 Redis 서버 연결을 확인합니다
func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// Set은 키-값을 저장합니다
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.rdb.Set(ctx, key, value, expiration).Err()
}

// Get은 키에 해당하는 값을 조회합니다
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

// Delete는 키를 삭제합니다
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

// Exists는 키의 존재 여부를 확인합니다
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	return c.rdb.Exists(ctx, keys...).Result()
}

// Expire는 키에 TTL을 설정합니다
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.rdb.Expire(ctx, key, expiration).Err()
}

// Increment는 숫자 값을 원자적으로 증가시킵니다
func (c *Client) Increment(ctx context.Context, key string) (int64, error) {
	return c.rdb.Incr(ctx, key).Result()
}

// Decrement는 숫자 값을 원자적으로 감소시킵니다
func (c *Client) Decrement(ctx context.Context, key string) (int64, error) {
	return c.rdb.Decr(ctx, key).Result()
}

// HSet은 해시 필드에 값을 저장합니다
func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) error {
	return c.rdb.HSet(ctx, key, values...).Err()
}

// HGet은 해시 필드의 값을 조회합니다
func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	return c.rdb.HGet(ctx, key, field).Result()
}

// HGetAll은 해시의 모든 필드와 값을 조회합니다
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.rdb.HGetAll(ctx, key).Result()
}

// LPush는 리스트의 왼쪽에 값을 추가합니다
func (c *Client) LPush(ctx context.Context, key string, values ...interface{}) error {
	return c.rdb.LPush(ctx, key, values...).Err()
}

// RPush는 리스트의 오른쪽에 값을 추가합니다
func (c *Client) RPush(ctx context.Context, key string, values ...interface{}) error {
	return c.rdb.RPush(ctx, key, values...).Err()
}

// LRange는 리스트의 범위를 조회합니다
func (c *Client) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.rdb.LRange(ctx, key, start, stop).Result()
}
