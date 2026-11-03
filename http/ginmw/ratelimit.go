package ginmw

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hktkhub/commonsdk/http/response"
	"github.com/hktkhub/commonsdk/ratelimit"
)

const rateLimitMessage = "too many requests"

// RateLimit 返回基于 ratelimit.Limiter 的 gin 限流中间件。
//
// keyFunc 决定限流维度，例如 IP、用户或它们的组合：
//
//	ginmw.RateLimit(limiter, func(c *gin.Context) string {
//		return c.ClientIP()
//	})
//
// 超限时返回 429 + 统一响应体，并终止后续 handler。
func RateLimit(limiter ratelimit.Limiter, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	if limiter == nil {
		panic("ginmw: nil rate limiter")
	}
	if keyFunc == nil {
		panic("ginmw: nil rate limit key func")
	}

	return func(c *gin.Context) {
		if !limiter.Allow(c.Request.Context(), keyFunc(c)) {
			c.AbortWithStatusJSON(
				http.StatusTooManyRequests,
				response.Error(http.StatusTooManyRequests, rateLimitMessage),
			)
			return
		}
		c.Next()
	}
}
