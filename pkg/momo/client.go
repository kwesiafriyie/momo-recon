package momo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client talks to the MTN MoMo Collections API.
type Client struct {
	baseURL         string
	subscriptionKey string
	apiUserID       string
	apiKey          string
	callbackURL     string
	targetEnv       string
	http            *http.Client
}

func NewClient(baseURL, subscriptionKey, apiUserID, apiKey, callbackURL, targetEnv string) *Client {
	return &Client{
		baseURL:         baseURL,
		subscriptionKey: subscriptionKey,
		apiUserID:       apiUserID,
		apiKey:          apiKey,
		callbackURL:     callbackURL,
		targetEnv:       targetEnv,
		http:            &http.Client{Timeout: 30 * time.Second},
	}
}

// RequestToPay initiates a Collections request. Returns the referenceId (UUID) used to poll status.
// The externalId should be set to the invoice reference_code so MoMo echoes it back in callbacks.
func (c *Client) RequestToPay(ctx context.Context, amount float64, currency, phone, externalId, referenceId string) error {
	body := map[string]any{
		"amount":     fmt.Sprintf("%.0f", amount),
		"currency":   currency,
		"externalId": externalId,
		"payer": map[string]string{
			"partyIdType": "MSISDN",
			"partyId":     phone,
		},
		"payerMessage": "Payment for invoice " + externalId,
		"payeeNote":    "Invoice " + externalId,
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("momo: get token: %w", err)
	}

	req, err := c.newRequest(ctx, http.MethodPost, "/collection/v1_0/requesttopay", body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Reference-Id", referenceId) // our UUID — becomes the poll key
	req.Header.Set("X-Callback-Url", c.callbackURL)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("momo: requesttopay: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("momo: requesttopay: unexpected status %d: %s", resp.StatusCode, b)
	}
	return nil
}

type TransactionStatus struct {
	Amount                 string `json:"amount"`
	Currency               string `json:"currency"`
	FinancialTransactionId string `json:"financialTransactionId"`
	ExternalId             string `json:"externalId"`
	Status                 string `json:"status"` // PENDING | SUCCESSFUL | FAILED
	Reason                 any    `json:"reason"`
}

// GetTransactionStatus polls the status of a previously initiated request.
func (c *Client) GetTransactionStatus(ctx context.Context, referenceId string) (*TransactionStatus, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("momo: get token: %w", err)
	}

	req, err := c.newRequest(ctx, http.MethodGet, "/collection/v1_0/requesttopay/"+referenceId, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("momo: getstatus: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("momo: getstatus: unexpected status %d: %s", resp.StatusCode, b)
	}

	var s TransactionStatus
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, fmt.Errorf("momo: getstatus: decode: %w", err)
	}
	return &s, nil
}

// --- internal helpers ---

type tokenResponse struct {
	AccessToken string `json:"access_token"`
}

func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/collection/token/", nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.apiUserID, c.apiKey)
	req.Header.Set("Ocp-Apim-Subscription-Key", c.subscriptionKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("momo: token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("momo: token: unexpected status %d: %s", resp.StatusCode, b)
	}

	var t tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return "", fmt.Errorf("momo: token: decode: %w", err)
	}
	return t.AccessToken, nil
}

func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		buf = bytes.NewBuffer(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Ocp-Apim-Subscription-Key", c.subscriptionKey)
	req.Header.Set("X-Target-Environment", c.targetEnv)
	return req, nil
}
