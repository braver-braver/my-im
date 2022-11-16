package cache

//go:generate mockgen -source=user_info_cache.go -destination=../../internal/mock/user_info_cache_mock.go  -package mock

import (
	"context"
	"fmt"
	"time"

	"github.com/go-eagle/eagle/pkg/cache"
	"github.com/go-eagle/eagle/pkg/encoding"
	"github.com/go-eagle/eagle/pkg/log"
	"github.com/go-eagle/eagle/pkg/redis"

	"user-service/internal/model"
)

const (
	// PrefixUserInfoCacheKey cache prefix
	PrefixUserInfoCacheKey = "user:info:%d"
)

// UserInfoCache define cache interface
type UserInfoCache interface {
	SetUserInfoCache(ctx context.Context, id int64, data *model.UserInfoModel, duration time.Duration) error
	GetUserInfoCache(ctx context.Context, id int64) (data *model.UserInfoModel, err error)
	MultiGetUserInfoCache(ctx context.Context, ids []int64) (map[string]*model.UserInfoModel, error)
	MultiSetUserInfoCache(ctx context.Context, data []*model.UserInfoModel, duration time.Duration) error
	DelUserInfoCache(ctx context.Context, id int64) error
}

// userInfoCache define cache struct
type userInfoCache struct {
	cache cache.Cache
}

// NewUserInfoCache new a cache
func NewUserInfoCache() UserInfoCache {
	jsonEncoding := encoding.JSONEncoding{}
	cachePrefix := ""
	return &userInfoCache{
		cache: cache.NewRedisCache(redis.RedisClient, cachePrefix, jsonEncoding, func() interface{} {
			return &model.UserInfoModel{}
		}),
	}
}

// GetUserInfoCacheKey get cache key
func (c *userInfoCache) GetUserInfoCacheKey(id int64) string {
	return fmt.Sprintf(PrefixUserInfoCacheKey, id)
}

// SetUserInfoCache write to cache
func (c *userInfoCache) SetUserInfoCache(ctx context.Context, id int64, data *model.UserInfoModel, duration time.Duration) error {
	if data == nil || id == 0 {
		return nil
	}
	cacheKey := c.GetUserInfoCacheKey(id)
	err := c.cache.Set(ctx, cacheKey, data, duration)
	if err != nil {
		return err
	}
	return nil
}

// GetUserInfoCache get from cache
func (c *userInfoCache) GetUserInfoCache(ctx context.Context, id int64) (data *model.UserInfoModel, err error) {
	cacheKey := c.GetUserInfoCacheKey(id)
	err = c.cache.Get(ctx, cacheKey, &data)
	if err != nil {
		log.WithContext(ctx).Warnf("get err from redis, err: %+v", err)
		return nil, err
	}
	return data, nil
}

// MultiGetUserInfoCache batch get cache
func (c *userInfoCache) MultiGetUserInfoCache(ctx context.Context, ids []int64) (map[string]*model.UserInfoModel, error) {
	var keys []string
	for _, v := range ids {
		cacheKey := c.GetUserInfoCacheKey(v)
		keys = append(keys, cacheKey)
	}

	// NOTE: 需要在这里make实例化，如果在返回参数里直接定义会报 nil map
	retMap := make(map[string]*model.UserInfoModel)
	err := c.cache.MultiGet(ctx, keys, retMap)
	if err != nil {
		return nil, err
	}
	return retMap, nil
}

// MultiSetUserInfoCache batch set cache
func (c *userInfoCache) MultiSetUserInfoCache(ctx context.Context, data []*model.UserInfoModel, duration time.Duration) error {
	valMap := make(map[string]interface{})
	for _, v := range data {
		cacheKey := c.GetUserInfoCacheKey(v.ID)
		valMap[cacheKey] = v
	}

	err := c.cache.MultiSet(ctx, valMap, duration)
	if err != nil {
		return err
	}
	return nil
}

// DelUserInfoCache delete cache
func (c *userInfoCache) DelUserInfoCache(ctx context.Context, id int64) error {
	cacheKey := c.GetUserInfoCacheKey(id)
	err := c.cache.Del(ctx, cacheKey)
	if err != nil {
		return err
	}
	return nil
}

// SetCacheWithNotFound set empty cache
func (c *userInfoCache) SetCacheWithNotFound(ctx context.Context, id int64) error {
	cacheKey := c.GetUserInfoCacheKey(id)
	err := c.cache.SetCacheWithNotFound(ctx, cacheKey)
	if err != nil {
		return err
	}
	return nil
}
