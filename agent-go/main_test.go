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

func TestAnyDeskExecutableCandidatesPrioritizeConfiguredPath(t *testing.T) {
	configured := `E:\\Apps\\AnyDesk\\AnyDesk.exe`

	candidates := anyDeskExecutableCandidates(configured)

	if candidates[0] != configured {
		t.Fatalf("expected configured executable first, got %q", candidates[0])
	}
}

func TestApplyReopenAttemptsWhenAnyDeskWasAlreadyClosed(t *testing.T) {
	called := false
	result := applyReopen(outcome{Matched: 0}, true, `E:\\Apps\\AnyDesk\\AnyDesk.exe`, func(string) bool {
		called = true
		return true
	})

	if !called {
		t.Fatal("expected AnyDesk opener to be called when reopening was requested")
	}
	if !result.ReopenAttempted {
		t.Fatal("expected reopenAttempted to be true")
	}
	if !result.Reopened {
		t.Fatal("expected reopened to report opener success")
	}
}
