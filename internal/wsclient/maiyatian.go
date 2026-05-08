package wsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/awfufu/gopick/internal/config"
	"github.com/awfufu/gopick/internal/domain"
	"github.com/awfufu/gopick/internal/maiyatian"
	"github.com/gorilla/websocket"
)

const wsURL = "wss://msg.maiyatian.com/acc"

type MaiyatianWSClient struct {
	logger  *slog.Logger
	client  maiyatian.Client
	config  config.MaiyatianConfig
	handler NewOrderHandler

	mu     sync.RWMutex
	status domain.WSStatus
}

type NewOrderHandler interface {
	HandleNewOrder(ctx context.Context, id string) error
}

type wsEnvelope struct {
	Seq  string          `json:"seq,omitempty"`
	Cmd  string          `json:"cmd"`
	Msg  string          `json:"msg,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

type wsLoginData struct {
	MerchantID string `json:"merchant_id"`
	AccountID  string `json:"account_id"`
	Shop       string `json:"shop"`
	City       string `json:"city"`
}

type notifyPayload struct {
	Type     string `json:"type"`
	Message  string `json:"msg"`
	ID       any    `json:"id"`
	Platform string `json:"platform"`
}

func NewMaiyatianWSClient(logger *slog.Logger, client maiyatian.Client, cfg config.MaiyatianConfig, handler NewOrderHandler) *MaiyatianWSClient {
	return &MaiyatianWSClient{
		logger:  logger,
		client:  client,
		config:  cfg,
		handler: handler,
		status:  domain.WSStatus{URL: wsURL},
	}
}

func (c *MaiyatianWSClient) Run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		if err := c.runOnce(ctx); err != nil {
			c.setDisconnected(err)
			c.logger.Error("maiyatian ws disconnected", "error", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

func (c *MaiyatianWSClient) Status() domain.WSStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}

func (c *MaiyatianWSClient) runOnce(ctx context.Context) error {
	contextInfo, err := c.client.GetOrderContext(ctx)
	if err != nil {
		return fmt.Errorf("get order context: %w", err)
	}

	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 15 * time.Second,
	}

	headers := http.Header{}
	headers.Set("Origin", "https://saas.maiyatian.com")
	headers.Set("User-Agent", c.config.UserAgent)

	conn, _, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		return fmt.Errorf("dial ws: %w", err)
	}
	defer conn.Close()

	if err := c.login(conn, contextInfo); err != nil {
		return err
	}

	now := time.Now().Format(time.RFC3339)
	c.mu.Lock()
	c.status.URL = wsURL
	c.status.Connected = true
	c.status.Authenticated = true
	c.status.MerchantID = contextInfo.MerchantID
	c.status.AccountID = contextInfo.AccountID
	c.status.LastConnectedAt = now
	c.status.LastAuthenticatedAt = now
	c.status.LastError = ""
	c.mu.Unlock()

	readErrCh := make(chan error, 1)
	go c.readLoop(conn, readErrCh)

	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-readErrCh:
			if err != nil {
				return err
			}
			return fmt.Errorf("ws read loop stopped")
		case <-heartbeatTicker.C:
			if err := c.writeJSON(conn, wsEnvelope{Cmd: "heartbeat", Data: json.RawMessage(`{}`)}); err != nil {
				return fmt.Errorf("send heartbeat: %w", err)
			}
			c.mu.Lock()
			c.status.LastHeartbeatAt = time.Now().Format(time.RFC3339)
			c.mu.Unlock()
		}
	}
}

func (c *MaiyatianWSClient) login(conn *websocket.Conn, contextInfo domain.OrderContext) error {
	payload := wsEnvelope{
		Cmd: "login",
		Data: mustMarshalRaw(wsLoginData{
			MerchantID: contextInfo.MerchantID,
			AccountID:  contextInfo.AccountID,
			Shop:       "0",
			City:       "0",
		}),
	}

	if err := c.writeJSON(conn, payload); err != nil {
		return fmt.Errorf("send login: %w", err)
	}

	_, message, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read login response: %w", err)
	}

	var response wsEnvelope
	if err := json.Unmarshal(message, &response); err != nil {
		return fmt.Errorf("decode login response: %w", err)
	}
	if response.Cmd != "login" || response.Msg != "Success" {
		return fmt.Errorf("login failed: cmd=%s msg=%s", response.Cmd, response.Msg)
	}

	c.mu.Lock()
	c.status.LastMessageCmd = response.Cmd
	c.status.LastMessageAt = time.Now().Format(time.RFC3339)
	c.mu.Unlock()

	return nil
}

func (c *MaiyatianWSClient) readLoop(conn *websocket.Conn, errCh chan<- error) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			errCh <- err
			return
		}

		var envelope wsEnvelope
		if err := json.Unmarshal(message, &envelope); err != nil {
			continue
		}

		c.mu.Lock()
		c.status.LastMessageCmd = envelope.Cmd
		c.status.LastMessageAt = time.Now().Format(time.RFC3339)
		c.mu.Unlock()

		if msg := formatEnvelopeLogMessage(envelope); msg != "" {
			c.logger.Info("maiyatian ws message", "cmd", envelope.Cmd, "msg", msg)
		}

		c.handleEnvelope(envelope)
	}
}

func (c *MaiyatianWSClient) handleEnvelope(envelope wsEnvelope) {
	if c.handler == nil || envelope.Cmd != "notify" {
		return
	}

	var payload notifyPayload
	if err := json.Unmarshal([]byte(envelope.Msg), &payload); err != nil {
		c.logger.Warn("failed to decode ws notify payload", "error", err, "msg", envelope.Msg)
		return
	}
	if payload.Type != "confirm" {
		return
	}

	orderID := strings.TrimSpace(asString(payload.ID))
	if orderID == "" {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := c.handler.HandleNewOrder(ctx, orderID); err != nil {
			c.logger.Error("failed to report new order", "orderId", orderID, "error", err)
			return
		}
		c.logger.Info("reported new order to myshop", "orderId", orderID)
	}()
}

func isHeartbeatSuccess(envelope wsEnvelope) bool {
	return envelope.Cmd == "heartbeat" && strings.EqualFold(strings.TrimSpace(envelope.Msg), "Success")
}

func formatEnvelopeLogMessage(envelope wsEnvelope) string {
	msg := strings.TrimSpace(envelope.Msg)
	if msg == "" || isHeartbeatSuccess(envelope) {
		return ""
	}

	if envelope.Cmd != "notify" {
		return msg
	}

	var payload notifyPayload
	if err := json.Unmarshal([]byte(msg), &payload); err != nil {
		return msg
	}

	decoded := strings.TrimSpace(payload.Message)
	if decoded == "" {
		return msg
	}

	if payload.Type != "" {
		return fmt.Sprintf("%s: %s", payload.Type, decoded)
	}

	return decoded
}

func (c *MaiyatianWSClient) writeJSON(conn *websocket.Conn, payload wsEnvelope) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, data)
}

func (c *MaiyatianWSClient) setDisconnected(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status.URL = wsURL
	c.status.Connected = false
	c.status.Authenticated = false
	c.status.ReconnectCount++
	if err != nil {
		c.status.LastError = err.Error()
	}
}

func mustMarshalRaw(value any) json.RawMessage {
	data, _ := json.Marshal(value)
	return data
}

func asString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case json.Number:
		return typed.String()
	default:
		return fmt.Sprint(value)
	}
}
