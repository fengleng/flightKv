package redis

import (
	"context"
	"encoding/json"
	"github.com/fengleng/flightKv"
	"strconv"
	"time"

	"github.com/fengleng/flightKv/store"
	"github.com/go-redis/redis/v8"
	"github.com/pingcap/errors"
	//"github.com/smallnest/rpcx/log"
)

// Register registers Redis to valkeyrie
func Register() {
	flightKv.AddStore(store.REDIS, New)

}

var (
	// ErrMultipleEndpointsUnsupported is thrown when there are
	// multiple endpoints specified for Redis
	ErrMultipleEndpointsUnsupported = errors.New("redis: does not support multiple endpoints")

	// ErrTLSUnsupported is thrown when tls config is given
	ErrTLSUnsupported = errors.New("redis does not support tls")

	// ErrAbortTryLock is thrown when a user stops trying to seek the lock
	// by sending a signal to the stop chan, this is used to verify if the
	// operation succeeded
	ErrAbortTryLock = errors.New("redis: lock operation aborted")
)

// Redis implements valkeyrie.Store interface with redis backend
type Redis struct {
	client *redis.Client
	codec  jsonCodec
}

func New(endpoints []string, options *store.Config) (*Redis, error) {
	var password string
	var dbIndex = 0
	if options != nil && options.TLS != nil {
		return nil, ErrTLSUnsupported
	}
	if len(endpoints) > 1 {
		return nil, ErrMultipleEndpointsUnsupported
	}
	if options != nil && options.PassWord != "" {
		password = options.PassWord
	}
	if options != nil && options.Bucket != "" {
		tmp, err := strconv.Atoi(options.Bucket)
		if err != nil {
			return nil, errors.Trace(err)
		}
		dbIndex = tmp
	}
	return newRedis(endpoints, password, dbIndex)
}

func newRedis(endpoints []string, password string, dbIndex int) (*Redis, error) {
	// TODO: use *redis.ClusterClient if we support miltiple endpoints
	client := redis.NewClient(&redis.Options{
		Addr:         endpoints[0],
		DialTimeout:  5 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		Password:     password,
		DB:           dbIndex,
	})

	// Listen to Keyspace events
	client.ConfigSet(context.Background(), "notify-keyspace-events", "KEA")

	return &Redis{
		client: client,
		codec:  jsonCodec{},
	}, nil
}

type jsonCodec struct{}

func (c jsonCodec) encode(kv *store.KVPair) (string, error) {
	b, err := json.Marshal(kv)
	return string(b), err
}

func (c jsonCodec) decode(b string, kv *store.KVPair) error {
	return json.Unmarshal([]byte(b), kv)
}
