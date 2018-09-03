package redis

import (
	"fmt"
	"strconv"

	"github.com/go-redis/redis"
	"github.com/mainflux/mainflux/things"
)

var _ things.ChannelCache = (*channelCache)(nil)

type channelCache struct {
	client *redis.Client
}

// NewChannelCache returns redis cache implementation.
func NewChannelCache(client *redis.Client) things.ChannelCache {
	return channelCache{client: client}
}

func (cc channelCache) Save(chanID, thingID uint64, thingKey string) error {
	if err := cc.client.Set(fmt.Sprintf("%s", thingKey), thingID, 0).Err(); err != nil {
		return err
	}

	return cc.client.SAdd(fmt.Sprintf("%d", chanID), thingKey).Err()
}

func (cc channelCache) Connected(chanID uint64, thingKey string) (uint64, error) {
	result, err := cc.client.SIsMember(fmt.Sprintf("%d", chanID), thingKey).Result()
	if !result || err != nil {
		return 0, things.ErrNotFound
	}

	id, err := cc.client.Get(thingKey).Result()
	if err != nil {
		return 0, err
	}

	thingID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0, err
	}

	return thingID, nil
}
