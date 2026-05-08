package myshop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/awfufu/gopick/internal/config"
	"github.com/awfufu/gopick/internal/domain"
)

type Client struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

type createListenedOrderRequest struct {
	ID                    string             `json:"id"`
	LogisticID            string             `json:"logisticId"`
	City                  int                `json:"city"`
	Platform              string             `json:"platform"`
	DailyPlatformSequence int                `json:"dailyPlatformSequence"`
	OrderNo               string             `json:"orderNo"`
	OrderTime             string             `json:"orderTime"`
	UserAddress           string             `json:"userAddress"`
	Longitude             float64            `json:"longitude"`
	Latitude              float64            `json:"latitude"`
	ExternalStatus        string             `json:"externalStatus"`
	DeliveryDeadline      string             `json:"deliveryDeadline"`
	DistanceKM            float64            `json:"distanceKm"`
	DistanceIsLinear      bool               `json:"distanceIsLinear"`
	ActualPaid            int                `json:"actualPaid"`
	PlatformCommission    int                `json:"platformCommission"`
	Items                 []domain.OrderItem `json:"items"`
}

func NewClient(cfg config.UploadConfig) *Client {
	return &Client{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:  strings.TrimSpace(cfg.APIKey),
		client:  &http.Client{Timeout: cfg.Timeout},
	}
}

func (c *Client) CreateListenedOrder(ctx context.Context, order domain.Order) error {
	request := createListenedOrderRequest{
		ID:                    order.ID,
		LogisticID:            order.LogisticID,
		City:                  order.City,
		Platform:              order.Platform,
		DailyPlatformSequence: order.DailyPlatformSequence,
		OrderNo:               order.OrderNo,
		OrderTime:             order.OrderTime,
		UserAddress:           order.UserAddress,
		Longitude:             order.Longitude,
		Latitude:              order.Latitude,
		ExternalStatus:        order.Status,
		DeliveryDeadline:      order.DeliveryDeadline,
		DistanceKM:            order.DistanceKM,
		DistanceIsLinear:      order.DistanceIsLinear,
		ActualPaid:            order.ActualPaid,
		PlatformCommission:    order.PlatformCommission,
		Items:                 order.Items,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return err
	}

	endpoint := c.baseURL + "/api/v1/api-key/listened-orders"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("myshop listened-orders returned status %d", resp.StatusCode)
	}

	return nil
}
