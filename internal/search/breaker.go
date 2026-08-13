package search

import (
	"sync"
	"time"
)

// breakerState is the classic three-state circuit breaker.
type breakerState int

const (
	stateClosed   breakerState = iota // healthy: requests flow to ES
	stateOpen                         // tripped: skip ES, go straight to fallback
	stateHalfOpen                     // probing: allow one trial request
)

// circuitBreaker guards the Elasticsearch path. After `threshold` consecutive
// failures it opens for `cooldown`, during which every call is routed straight
// to the Postgres fallback without touching ES. After the cooldown it allows a
// single trial request (half-open); success closes it, failure re-opens it.
//
// This keeps p99 latency low when ES is down: we stop waiting on timeouts and
// serve from Postgres immediately.
type circuitBreaker struct {
	mu           sync.Mutex
	state        breakerState
	failures     int
	threshold    int
	cooldown     time.Duration
	openedAt     time.Time
	now          func() time.Time // injectable clock for tests
}

func newCircuitBreaker(threshold int, cooldown time.Duration) *circuitBreaker {
	if threshold <= 0 {
		threshold = 5
	}
	if cooldown <= 0 {
		cooldown = 15 * time.Second
	}
	return &circuitBreaker{
		state:     stateClosed,
		threshold: threshold,
		cooldown:  cooldown,
		now:       time.Now,
	}
}

// Allow reports whether a request may attempt the ES path right now.
func (cb *circuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case stateClosed:
		return true
	case stateOpen:
		if cb.now().Sub(cb.openedAt) >= cb.cooldown {
			cb.state = stateHalfOpen
			return true // single trial
		}
		return false
	case stateHalfOpen:
		// Only one trial at a time; further callers use the fallback until the
		// trial resolves.
		return false
	default:
		return true
	}
}

// Success records a successful ES call, closing the breaker.
func (cb *circuitBreaker) Success() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.state = stateClosed
}

// Failure records a failed ES call, tripping the breaker at the threshold.
func (cb *circuitBreaker) Failure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == stateHalfOpen {
		cb.trip()
		return
	}
	cb.failures++
	if cb.failures >= cb.threshold {
		cb.trip()
	}
}

func (cb *circuitBreaker) trip() {
	cb.state = stateOpen
	cb.openedAt = cb.now()
}

// State returns a human-readable state for metrics/logging.
func (cb *circuitBreaker) State() string {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	switch cb.state {
	case stateOpen:
		return "open"
	case stateHalfOpen:
		return "half_open"
	default:
		return "closed"
	}
}
