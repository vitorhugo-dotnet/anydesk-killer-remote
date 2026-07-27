package main

import "testing"

func TestRedisOptionsUsesConfiguredDatabase(t *testing.T) {
	var c config
	c.Redis.Database = 3

	options := redisOptions(c, "127.0.0.1:6379")

	if options.DB != 3 {
		t.Fatalf("expected Redis database 3, got %d", options.DB)
	}
}

func TestRedisOptionsDefaultsToDatabaseZero(t *testing.T) {
	var c config

	options := redisOptions(c, "127.0.0.1:6379")

	if options.DB != 0 {
		t.Fatalf("expected Redis database 0, got %d", options.DB)
	}
}
