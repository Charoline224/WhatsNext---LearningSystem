package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisQueue struct {
	client *redis.Client
	name   string
}

func NewRedisQueue(address, password, name string) *RedisQueue {
	return &RedisQueue{client: redis.NewClient(&redis.Options{Addr: address, Password: password}), name: name}
}
func (q *RedisQueue) Ping(ctx context.Context) error {
	if err := q.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	return nil
}
func (q *RedisQueue) Enqueue(ctx context.Context, jobID string) error {
	if err := q.client.LPush(ctx, q.name, jobID).Err(); err != nil {
		return fmt.Errorf("enqueue job: %w", err)
	}
	return nil
}
func (q *RedisQueue) Dequeue(ctx context.Context, timeout time.Duration) (string, error) {
	result, err := q.client.BRPop(ctx, timeout, q.name).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("dequeue job: %w", err)
	}
	if len(result) != 2 {
		return "", nil
	}
	return result[1], nil
}
func (q *RedisQueue) Close() error { return q.client.Close() }
