package http

import (
	"errors"
	"log"
	"net/http"

	"order-service/internal/domain"
	"order-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	uc *usecase.OrderUseCase
}

func NewOrderHandler(uc *usecase.OrderUseCase) *OrderHandler {
	return &OrderHandler{uc: uc}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	//бонус
	var idempotencyKey *string
	if key := c.GetHeader("Idempotency-Key"); key != "" {
		idempotencyKey = &key
	}

	order, err := h.uc.CreateOrder(c.Request.Context(), req.CustomerID, req.ItemName, req.Amount, idempotencyKey)
	if err != nil {
		h.handleError(c, err, order)
		return
	}

	c.JSON(http.StatusCreated, toOrderResponse(order))
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")

	order, err := h.uc.GetOrder(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err, nil)
		return
	}

	c.JSON(http.StatusOK, toOrderResponse(order))
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	id := c.Param("id")

	order, err := h.uc.CancelOrder(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err, nil)
		return
	}

	c.JSON(http.StatusOK, toOrderResponse(order))
}

func (h *OrderHandler) GetOrdersByCustomer(c *gin.Context) {
	customerID := c.Param("customer_id")

	orders, err := h.uc.GetOrdersByCustomer(c.Request.Context(), customerID)
	if err != nil {
		h.handleError(c, err, nil)
		return
	}

	response := make([]OrderResponse, 0, len(orders))
	for _, order := range orders {
		response = append(response, toOrderResponse(order))
	}

	c.JSON(http.StatusOK, response)
}

func toOrderResponse(order *domain.Order) OrderResponse {
	return OrderResponse{
		ID:         order.ID,
		CustomerID: order.CustomerID,
		ItemName:   order.ItemName,
		Amount:     order.Amount,
		Status:     order.Status,
		CreatedAt:  order.CreatedAt,
	}
}

func (h *OrderHandler) handleError(c *gin.Context, err error, order *domain.Order) {
	log.Printf("[HANDLER ERROR] %v", err)

	switch {
	case errors.Is(err, domain.ErrOrderNotFound):
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "order not found"})

	case errors.Is(err, domain.ErrInvalidAmount):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "amount must be greater than zero"})

	case errors.Is(err, domain.ErrCancelNotAllowed):
		c.JSON(http.StatusConflict, ErrorResponse{Error: "only pending orders can be cancelled"})

	case errors.Is(err, domain.ErrInvalidCustomerID):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid customer id"})

	case errors.Is(err, domain.ErrPaymentServiceUnavailable):
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "payment service unavailable"})

	case errors.Is(err, domain.ErrPaymentDeclined):
		if order != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": "payment declined: amount exceeds limit",
				"order": toOrderResponse(order),
			})
		} else {
			c.JSON(http.StatusUnprocessableEntity, ErrorResponse{Error: "payment declined"})
		}

	default:
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}
}
