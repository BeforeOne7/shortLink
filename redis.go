package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/pilu/go-base62"
	"time"
)

const (
	// URLIDKEY 全局自增器
	URLIDKEY = "next.url.id"
	// ShortlinkKey 映射了短地址和长地址之间的关系
	ShortlinkKey = "shortlink:%s:url"
	// URLHashKey 映射了长地址的Hash值
	URLHashKey = "urlhash:%s:url"
	// ShortLinkDetailKey 短地址详情
	ShortLinkDetailKey = "shortlink:%s:detail"
)

type RedisClient struct {
	cli *redis.Client
}

type URLDetail struct {
	URL                 string
	CreatedAt           string
	ExpirationInMinutes time.Duration
}

func NewRedisClient(addr, password string, db int) *RedisClient {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	if _, err := client.Ping(context.Background()).Result(); err != nil {
		panic(err)
	}

	return &RedisClient{client}
}

// Shorten 将URL转换成短地址
func (r *RedisClient) Shorten(url string, exp int64) (string, error) {
	// 生成URL的HASH
	h := toSHA1(url)

	// 先在缓存中找
	d, err := r.cli.Get(context.Background(), fmt.Sprintf(URLHashKey, h)).Result()
	if err == redis.Nil {

	} else if err != nil {
		return "", err
	} else {
		if d == "{}" {

		} else {
			return d, err
		}
	}

	// 第一次增加则自增
	err = r.cli.Incr(context.Background(), URLIDKEY).Err()
	if err != nil {
		return "", err
	}

	id, err := r.cli.Get(context.Background(), URLIDKEY).Int()
	if err != nil {
		return "", err
	}
	eid := base62.Encode(id)

	err = r.cli.Set(context.Background(), fmt.Sprintf(ShortLinkDetailKey, eid), url, time.Minute*time.Duration(exp)).Err()
	if err != nil {
		return "", err
	}

	err = r.cli.Set(context.Background(), fmt.Sprintf(URLHashKey, h), eid, time.Minute*time.Duration(exp)).Err()
	if err != nil {
		return "", err
	}
	detail, err := json.Marshal(&URLDetail{
		URL:                 url,
		CreatedAt:           time.Now().String(),
		ExpirationInMinutes: time.Duration(exp),
	})
	if err != nil {
		return "", err
	}
	err = r.cli.Set(context.Background(), fmt.Sprintf(ShortLinkDetailKey, eid), detail, time.Minute*time.Duration(exp)).Err()
	if err != nil {
		return "", nil
	}
	return eid, err
}

func toSHA1(url string) interface{} {
	h := sha1.New()
	h.Write([]byte(url))
	return hex.EncodeToString(h.Sum(nil))
}

func (r *RedisClient) ShortLinkInfo(eid string) (URLDetail, error) {
	var detailInfo URLDetail
	detail, err := r.cli.Get(context.Background(), fmt.Sprintf(ShortLinkDetailKey, eid)).Result()
	if err == redis.Nil {
		return detailInfo, StatusError{
			Code: 404,
			Err:  errors.New("unknown short link"),
		}
	} else if err != nil {
		return detailInfo, err
	}
	err = json.Unmarshal([]byte(detail), &detailInfo)
	if err != nil {
		return detailInfo, err
	}
	return detailInfo, nil
}

func (r *RedisClient) UnShorten(encodeId string) (string, error) {
	url, err := r.cli.Get(context.Background(), fmt.Sprintf(ShortlinkKey, encodeId)).Result()
	if err == redis.Nil {
		return "", StatusError{
			Code: 404,
			Err:  errors.New("unknown short link"),
		}
	} else if err != nil {
		return "", err
	}

	return url, nil
}
