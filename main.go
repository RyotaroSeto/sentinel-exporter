package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

var (
	redisConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "myapp_redis_connections",
		Help: "The number of connections to Redis",
	})
)

func getMetrics(rdb *redis.Client) {
	val := rdb.PoolStats().TotalConns
	redisConnections.Set(float64(val))
}

func recordMetrics(rdb *redis.Client) {
	go func() {
		for {
			redisConnections.Inc()
			getMetrics(rdb)
			time.Sleep(2 * time.Second)
		}
	}()
}

func main() {
	redisSentinelHost := os.Getenv("REDIS_SENTINEL_HOST")
	if redisSentinelHost == "" {
		log.Fatal("REDIS_SENTINEL_HOST is not set")
	}

	redisSentinelPortStr := os.Getenv("REDIS_SENTINEL_PORT")
	if redisSentinelPortStr == "" {
		log.Fatal("REDIS_SENTINEL_PORT is not set")
	}

	redisSentinelPort, err := strconv.Atoi(redisSentinelPortStr)
	if err != nil {
		log.Fatalf("Invalid REDIS_SENTINEL_PORT: %s", err)
	}

	redisMasterName := os.Getenv("REDIS_MASTER_NAME")
	if redisMasterName == "" {
		log.Fatal("REDIS_MASTER_NAME is not set")
	}

	rdb := redis.NewFailoverClient(&redis.FailoverOptions{
		SentinelAddrs: []string{fmt.Sprintf("%s:%d", redisSentinelHost, redisSentinelPort)},
		MasterName:    redisMasterName,
		DialTimeout:   5 * time.Second,
		ReadTimeout:   5 * time.Second,
		WriteTimeout:  5 * time.Second,
	})

	recordMetrics(rdb)

	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(":9478", nil); err != nil {
		log.Fatal("Failed to start exporter:", err)
	}
}
