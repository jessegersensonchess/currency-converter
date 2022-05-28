package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"time"
)

var ctx = context.Background()

// use configuration management
const redisTTL = 3600

func redisSet(key string, val string) {
	Rdb := redis.NewClient(&redis.Options{
		// todo: replace hardcoded values
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	// TTL is
	err := Rdb.Set(ctx, key, val, redisTTL*time.Second).Err()
	if err != nil {
		println("error:", err)
	}
	return
}

func redisGet(key string) (string, error) {
	Rdb := redis.NewClient(&redis.Options{
		// todo: replace hardcoded values
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	val, err := Rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		fmt.Printf("key=%v does not exist in redis\n", key)
	} else if err != nil {
		println("error:", err)
	} else {
		//fmt.Println(key, val)
	}

	return val, err
}
