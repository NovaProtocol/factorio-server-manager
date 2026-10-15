package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLimiterAllowsUpToTheLimitThenBlocks(t *testing.T) {
	limiter := newAttemptLimiter(3, time.Minute)

	for attempt := 0; attempt < 3; attempt++ {
		allowed, _ := limiter.allow("1.2.3.4|admin")
		require.True(t, allowed, "attempt %d should be allowed", attempt)
		limiter.failure("1.2.3.4|admin")
	}

	allowed, wait := limiter.allow("1.2.3.4|admin")
	assert.False(t, allowed, "the fourth attempt is over the limit")
	assert.Greater(t, wait, time.Duration(0), "a blocked caller is told how long to wait")
}

func TestLimiterWindowExpiryReopensTheDoor(t *testing.T) {
	limiter := newAttemptLimiter(1, time.Minute)
	now := time.Now()
	limiter.now = func() time.Time { return now }

	limiter.failure("1.2.3.4|admin")
	allowed, _ := limiter.allow("1.2.3.4|admin")
	require.False(t, allowed)

	now = now.Add(time.Minute + time.Second)
	allowed, _ = limiter.allow("1.2.3.4|admin")
	assert.True(t, allowed, "a fresh window allows a new attempt")
}

func TestLimiterSuccessClearsTheKey(t *testing.T) {
	limiter := newAttemptLimiter(2, time.Minute)

	limiter.failure("1.2.3.4|admin")
	limiter.failure("1.2.3.4|admin")
	limiter.success("1.2.3.4|admin")

	allowed, _ := limiter.allow("1.2.3.4|admin")
	assert.True(t, allowed, "a successful login forgets the failures")
}

func TestLimiterKeysAreIndependent(t *testing.T) {
	limiter := newAttemptLimiter(1, time.Minute)

	limiter.failure("1.2.3.4|admin")

	allowed, _ := limiter.allow("1.2.3.4|someone-else")
	assert.True(t, allowed, "one name being guessed does not lock out another")

	allowed, _ = limiter.allow("5.6.7.8|admin")
	assert.True(t, allowed, "one address being throttled does not lock out another")
}

func TestLimiterSweepForgetsExpiredWindows(t *testing.T) {
	limiter := newAttemptLimiter(5, time.Minute)
	now := time.Now()
	limiter.now = func() time.Time { return now }

	limiter.failure("1.2.3.4|admin")
	require.Len(t, limiter.records, 1)

	now = now.Add(2 * time.Minute)
	limiter.allow("9.9.9.9|admin")

	assert.NotContains(t, limiter.records, "1.2.3.4|admin", "an expired window is dropped rather than kept")
}

func TestLoginThrottleKeySplitsAddressAndName(t *testing.T) {
	request, err := http.NewRequest("POST", "/api/login", nil)
	require.NoError(t, err)
	request.RemoteAddr = "10.0.0.7:41234"

	assert.Equal(t, "10.0.0.7|admin", loginThrottleKey(request, "admin"))

	request.RemoteAddr = "unparseable"
	assert.Equal(t, "unparseable|admin", loginThrottleKey(request, "admin"))
}
