package api

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// Login is the only unauthenticated endpoint, and the only place an anonymous
// caller can spend server CPU (bcrypt) guessing at a password. It is therefore
// the one place that is throttled: after a handful of failures inside the window
// the caller is answered with 429 without touching the password store.
//
// The limiter is deliberately in-memory: a restart clears it, a single manager
// process is the whole deployment, and a wrong guess costs a cluster of bcrypt
// hashes rather than a table write.
const (
	loginAttemptLimit  = 8
	loginAttemptWindow = 5 * time.Minute
)

type attemptRecord struct {
	count int
	first time.Time
}

type attemptLimiter struct {
	mutex     sync.Mutex
	records   map[string]*attemptRecord
	limit     int
	window    time.Duration
	now       func() time.Time
	lastSweep time.Time
}

var loginLimiter = newAttemptLimiter(loginAttemptLimit, loginAttemptWindow)

func newAttemptLimiter(limit int, window time.Duration) *attemptLimiter {
	return &attemptLimiter{
		records: make(map[string]*attemptRecord),
		limit:   limit,
		window:  window,
		now:     time.Now,
	}
}

// allow reports whether this key may try a password now, and how long to wait
// when it may not.
func (limiter *attemptLimiter) allow(key string) (bool, time.Duration) {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	limiter.sweep()

	record, ok := limiter.records[key]
	if !ok {
		return true, 0
	}

	elapsed := limiter.now().Sub(record.first)
	if elapsed >= limiter.window {
		delete(limiter.records, key)
		return true, 0
	}

	if record.count >= limiter.limit {
		return false, limiter.window - elapsed
	}

	return true, 0
}

// failure records one wrong password for that key.
func (limiter *attemptLimiter) failure(key string) {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	record, ok := limiter.records[key]
	if !ok || limiter.now().Sub(record.first) >= limiter.window {
		limiter.records[key] = &attemptRecord{count: 1, first: limiter.now()}
		return
	}

	record.count++
}

// success clears the key, so a legitimate user is never punished for someone
// else guessing their name from the same address.
func (limiter *attemptLimiter) success(key string) {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	delete(limiter.records, key)
}

// sweep drops windows that have expired, so the map cannot grow forever with
// addresses that tried once and left.
func (limiter *attemptLimiter) sweep() {
	now := limiter.now()
	if now.Sub(limiter.lastSweep) < limiter.window {
		return
	}

	for key, record := range limiter.records {
		if now.Sub(record.first) >= limiter.window {
			delete(limiter.records, key)
		}
	}
	limiter.lastSweep = now
}

// loginThrottleKey ties an attempt to the caller's address and the name it is
// guessing at, so one noisy address cannot lock out a different user.
func loginThrottleKey(r *http.Request, username string) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	return host + "|" + username
}

// tooManyAttempts writes the 429 the login handler returns when throttled.
func tooManyAttempts(w http.ResponseWriter, wait time.Duration) {
	seconds := int(wait.Seconds()) + 1
	w.Header().Set("Retry-After", fmt.Sprintf("%d", seconds))
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(http.StatusTooManyRequests)
	fmt.Fprintf(w, "too many login attempts, try again in %d seconds", seconds)
}
