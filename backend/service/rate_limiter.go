package service

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu      sync.Mutex
	entries map[string][]time.Time
	maxKeys int
}

func NewRateLimiter(maxKeys int) *RateLimiter {
	return &RateLimiter{
		entries: map[string][]time.Time{},
		maxKeys: maxKeys,
	}
}

func (l *RateLimiter) Allow(key string, limit int) bool {
	now := time.Now()
	cutoff := now.Add(-time.Minute)

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamps := l.entries[key]
	filtered := timestamps[:0]
	for _, timestamp := range timestamps {
		if timestamp.After(cutoff) {
			filtered = append(filtered, timestamp)
		}
	}

	if len(filtered) >= limit {
		l.entries[key] = filtered
		return false
	}

	if _, exists := l.entries[key]; !exists && len(l.entries) >= l.maxKeys {
		for existingKey := range l.entries {
			delete(l.entries, existingKey)
			break
		}
	}

	l.entries[key] = append(filtered, now)
	return true
}

func (l *RateLimiter) PurgeExpired() {
	cutoff := time.Now().Add(-time.Minute)

	l.mu.Lock()
	defer l.mu.Unlock()

	for key, timestamps := range l.entries {
		var keep []time.Time
		for _, timestamp := range timestamps {
			if timestamp.After(cutoff) {
				keep = append(keep, timestamp)
			}
		}
		if len(keep) == 0 {
			delete(l.entries, key)
		} else {
			l.entries[key] = keep
		}
	}
}
