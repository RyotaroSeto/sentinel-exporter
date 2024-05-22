package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/context"
)

var (
	redisAddr             string
	redisPassword         string
	scrapeInterval        int
	metricsPort           int
	debug                 bool
	masterInfoGauge       *prometheus.GaugeVec
	sentinelsCurrentGauge *prometheus.GaugeVec
)

func init() {
	flag.StringVar(&redisAddr, "addr", "localhost:26379", "Redis Sentinel address")
	flag.StringVar(&redisPassword, "password", "", "Redis password")
	flag.IntVar(&scrapeInterval, "interval", 30, "Scrape interval in seconds")
	flag.IntVar(&metricsPort, "metrics-port", 9478, "Metrics port")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.Parse()

	if debug {
		log.SetFlags(log.Lshortfile | log.LstdFlags)
	}

	masterInfoGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sentinel_master_info",
			Help: "Basic master information",
		},
		[]string{"master_name", "master_ip"},
	)

	sentinelsCurrentGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sentinel_sentinels_current",
			Help: "Number of running sentinels",
		},
		[]string{"master_name", "state"},
	)

	prometheus.MustRegister(masterInfoGauge)
	prometheus.MustRegister(sentinelsCurrentGauge)
}

func main() {
	go func() {
		for {
			collectMetrics()
			time.Sleep(time.Duration(scrapeInterval) * time.Second)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Printf("Starting server on :%d", metricsPort)
	log.Fatal(http.ListenAndServe(":9478", nil))
}

func collectMetrics() {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
	})

	masters, err := client.SentinelMasters(ctx).Result()
	if err != nil {
		log.Printf("Error fetching sentinel masters: %v", err)
		return
	}

	for _, master := range masters {
		masterName := master["name"]
		masterIP := master["ip"]

		masterInfoGauge.With(prometheus.Labels{
			"master_name": masterName,
			"master_ip":   masterIP,
		}).Set(1)

		sentinels, err := client.SentinelSentinels(ctx, masterName).Result()
		if err != nil {
			log.Printf("Error fetching sentinels for master %s: %v", masterName, err)
			continue
		}

		up, down := 0, 0
		for _, sentinel := range sentinels {
			if sentinel["is_disconnected"] == "0" {
				up++
			} else {
				down++
			}
		}

		sentinelsCurrentGauge.With(prometheus.Labels{
			"master_name": masterName,
			"state":       "up",
		}).Set(float64(up))
		sentinelsCurrentGauge.With(prometheus.Labels{
			"master_name": masterName,
			"state":       "down",
		}).Set(float64(down))
	}
}
