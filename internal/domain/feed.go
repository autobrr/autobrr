package domain

import "time"

type FeedCacheRepo interface {
	Get(bucket string, key string) ([]byte, error)
	Exists(bucket string, key string) (bool, error)
	Put(bucket string, key string, val []byte, ttl time.Duration) error
	Delete(bucket string, key string) error
}
