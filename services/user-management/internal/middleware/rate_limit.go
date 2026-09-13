package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"github.com/jayedbinnazir/golang-saas.git/internal/httpx"
)

// RateLimit throttles each client IP to roughly rps requests per second with a
// short burst. Visitors that go quiet for 5 minutes are forgotten.
func RateLimit(rps float64, burst int) gin.HandlerFunc {
	limiter := newIPLimiter(rate.Limit(rps), burst)
	go limiter.cleanupLoop()

	return func(c *gin.Context) {
		if !limiter.get(c.ClientIP()).Allow() {
			httpx.RenderError(c, httpx.TooManyRequests("too many requests, slow down"))
			return
		}
		c.Next()
	}
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    rate.Limit
	burst    int
}

func newIPLimiter(limit rate.Limit, burst int) *ipLimiter {
	return &ipLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		burst:    burst,
	}
}

func (l *ipLimiter) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	v, ok := l.visitors[ip]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(l.limit, l.burst)}
		l.visitors[ip] = v
	}
	v.lastSeen = time.Now()
	return v.limiter
}

func (l *ipLimiter) cleanupLoop() {
	for range time.Tick(time.Minute) {
		l.mu.Lock()
		for ip, v := range l.visitors {
			if time.Since(v.lastSeen) > 5*time.Minute {
				delete(l.visitors, ip)
			}
		}
		l.mu.Unlock()
	}
}
