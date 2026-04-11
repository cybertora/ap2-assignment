package http

import "github.com/gin-gonic/gin"

// NewRouter — НЕ ИЗМЕНЁН из Assignment 1.
// REST API остаётся тем же для внешних клиентов.
func NewRouter(handler *OrderHandler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.POST("/orders", handler.CreateOrder)
	r.GET("/orders/:id", handler.GetOrder)
	r.PATCH("/orders/:id/cancel", handler.CancelOrder)

	return r
}
