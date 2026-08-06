package cache

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type Store struct {
	rdb *redis.Client
	mu  sync.Mutex
	mem map[string]memEntry
}

type memEntry struct {
	val []byte
	exp time.Time
}

func New(redisURL string) *Store {
	s := &Store{mem: map[string]memEntry{}}
	if redisURL == "" {
		return s
	}
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return s
	}
	rdb := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return s
	}
	s.rdb = rdb
	return s
}

func (s *Store) EnabledRedis() bool {
	return s != nil && s.rdb != nil
}

func (s *Store) Get(ctx context.Context, key string) ([]byte, bool) {
	if s == nil {
		return nil, false
	}
	if s.rdb != nil {
		b, err := s.rdb.Get(ctx, key).Bytes()
		if err == nil {
			return b, true
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.mem[key]
	if !ok || time.Now().After(e.exp) {
		if ok {
			delete(s.mem, key)
		}
		return nil, false
	}
	return append([]byte(nil), e.val...), true
}

func (s *Store) Set(ctx context.Context, key string, val []byte, ttl time.Duration) {
	if s == nil {
		return
	}
	if s.rdb != nil {
		_ = s.rdb.Set(ctx, key, val, ttl).Err()
	}
	s.mu.Lock()
	s.mem[key] = memEntry{val: append([]byte(nil), val...), exp: time.Now().Add(ttl)}
	s.mu.Unlock()
}

func (s *Store) Del(ctx context.Context, keys ...string) {
	if s == nil || len(keys) == 0 {
		return
	}
	if s.rdb != nil {
		_ = s.rdb.Del(ctx, keys...).Err()
	}
	s.mu.Lock()
	for _, k := range keys {
		delete(s.mem, k)
	}
	s.mu.Unlock()
}

func (s *Store) DelByPrefix(ctx context.Context, prefix string) {
	if s == nil || prefix == "" {
		return
	}
	if s.rdb != nil {
		var cursor uint64
		for {
			keys, next, err := s.rdb.Scan(ctx, cursor, prefix+"*", 50).Result()
			if err != nil {
				break
			}
			if len(keys) > 0 {
				_ = s.rdb.Del(ctx, keys...).Err()
			}
			cursor = next
			if cursor == 0 {
				break
			}
		}
	}
	s.mu.Lock()
	for k := range s.mem {
		if strings.HasPrefix(k, prefix) {
			delete(s.mem, k)
		}
	}
	s.mu.Unlock()
}
