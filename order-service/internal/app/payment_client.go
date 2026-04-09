package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type PaymentClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewPaymentClient(baseURL string) *PaymentClient {
	return &PaymentClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 2 * time.Second, // REQUIRED: max 2 seconds timeout
		},
	}
}

type paymentRequest struct {
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
}

type paymentResponse struct {
	ID            string `json:"id"`
	OrderID       string `json:"order_id"`
	TransactionID string `json:"transaction_id"`
	Amount        int64  `json:"amount"`
	Status        string `json:"status"`
}

func (pc *PaymentClient) AuthorizePayment(ctx context.Context, orderID string, amount int64) (string, string, error) {
	reqBody := paymentRequest{
		OrderID: orderID,
		Amount:  amount,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal payment request: %w", err)
	}

	url := fmt.Sprintf("%s/payments", pc.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", "", fmt.Errorf("failed to create payment request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	log.Printf("[INFO] calling Payment Service: POST %s (order_id=%s, amount=%d)", url, orderID, amount)

	resp, err := pc.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("payment service call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read payment response: %w", err)
	}

	if resp.StatusCode >= 400 {
		log.Printf("[WARN] Payment Service returned status %d: %s", resp.StatusCode, string(respBody))

		var payResp paymentResponse
		if json.Unmarshal(respBody, &payResp) == nil && payResp.Status == "Declined" {
			return payResp.TransactionID, "Declined", nil
		}

		return "", "", fmt.Errorf("payment service error: status %d", resp.StatusCode)
	}

	var payResp paymentResponse
	if err := json.Unmarshal(respBody, &payResp); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal payment response: %w", err)
	}

	log.Printf("[INFO] Payment Service response: transaction_id=%s status=%s", payResp.TransactionID, payResp.Status)

	return payResp.TransactionID, payResp.Status, nil
}
