package http

import (
	"errors"
	"log"
	"net/http"

	"payment-service/internal/domain"
	"payment-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	uc *usecase.PaymentUseCase
}

func NewPaymentHandler(uc *usecase.PaymentUseCase) *PaymentHandler {
	return &PaymentHandler{uc: uc}
}

func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	payment, err := h.uc.AuthorizePayment(c.Request.Context(), req.OrderID, req.Amount)
	if err != nil {
		h.handleError(c, err)
		return
	}

	statusCode := http.StatusCreated
	if payment.Status == domain.StatusDeclined {
		statusCode = http.StatusUnprocessableEntity
	}

	c.JSON(statusCode, toPaymentResponse(payment))
}

func (h *PaymentHandler) GetPaymentByOrderID(c *gin.Context) {
	orderID := c.Param("order_id")

	payment, err := h.uc.GetPaymentByOrderID(c.Request.Context(), orderID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toPaymentResponse(payment))
}

func (h *PaymentHandler) ListPayments(c *gin.Context) {
	status := c.Query("status")

	payments, err := h.uc.ListPayments(c.Request.Context(), status)
	if err != nil {
		h.handleError(c, err)
		return
	}

	var response []PaymentResponse
	for _, p := range payments {
		response = append(response, toPaymentResponse(p))
	}

	if response == nil {
		response = []PaymentResponse{}
	}

	c.JSON(http.StatusOK, gin.H{
		"payments": response,
		"total":    len(response),
	})
}

func toPaymentResponse(payment *domain.Payment) PaymentResponse {
	return PaymentResponse{
		ID:            payment.ID,
		OrderID:       payment.OrderID,
		TransactionID: payment.TransactionID,
		Amount:        payment.Amount,
		Status:        payment.Status,
		CreatedAt:     payment.CreatedAt,
	}
}

func (h *PaymentHandler) handleError(c *gin.Context, err error) {
	log.Printf("[HANDLER ERROR] %v", err)

	switch {
	case errors.Is(err, domain.ErrPaymentNotFound):
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "payment not found"})

	case errors.Is(err, domain.ErrInvalidAmount):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "amount must be greater than zero"})

	case errors.Is(err, domain.ErrAmountExceedsLimit):
		c.JSON(http.StatusUnprocessableEntity, ErrorResponse{Error: "amount exceeds maximum limit"})

	default:
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}
}
