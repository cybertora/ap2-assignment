package http

import "github.com/gin-gonic/gin"

func NewRouter(handler *PaymentHandler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.POST("/payments", handler.CreatePayment)
	r.GET("/payments/:order_id", handler.GetPaymentByOrderID)

	return r
}
