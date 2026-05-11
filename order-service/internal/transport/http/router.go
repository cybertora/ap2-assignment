package http

import (
	"order-service/internal/transport/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// NewRouter — теперь принимает Redis и параметры rate limit.
// Если rateLimitEnabled=false или rdb==nil — middleware не подключается.
func NewRouter(handler *OrderHandler, rdb *redis.Client, rateLimitEnabled bool, rpm int) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	if rateLimitEnabled && rdb != nil {
		r.Use(middleware.NewRateLimiter(rdb, rpm))
	}

	r.POST("/orders", handler.CreateOrder)
	r.GET("/orders/payments", handler.ListPayments)
	r.GET("/orders/:id", handler.GetOrder)
	r.PATCH("/orders/:id/cancel", handler.CancelOrder)

	return r
}
