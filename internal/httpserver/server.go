package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/awfufu/gopick/internal/config"
	"github.com/awfufu/gopick/internal/domain"
	"github.com/awfufu/gopick/internal/service"
)

type wsStatusProvider interface {
	Status() domain.WSStatus
}

type Server struct {
	server       *http.Server
	logger       *slog.Logger
	orderService *service.OrderService
	httpConfig   config.HTTPConfig
	wsStatus     wsStatusProvider
}

type statusResponse struct {
	Service string          `json:"service"`
	Status  string          `json:"status"`
	Now     string          `json:"now"`
	Routes  []string        `json:"routes"`
	WS      domain.WSStatus `json:"ws"`
}

func New(cfg config.HTTPConfig, logger *slog.Logger, orderService *service.OrderService, wsStatus wsStatusProvider) *Server {
	s := &Server{
		logger:       logger,
		orderService: orderService,
		httpConfig:   cfg,
		wsStatus:     wsStatus,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /status", s.handleStatus)
	mux.HandleFunc("GET /order-context", s.handleOrderContext)
	mux.HandleFunc("GET /list-orders", s.handleListOrders)
	mux.HandleFunc("GET /list-orders/{status}", s.handleListOrders)
	mux.HandleFunc("GET /all-orders", s.handleAllOrders)
	mux.HandleFunc("GET /all-orders/{date}", s.handleAllOrders)

	s.server = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:           s.logRequests(mux),
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return s
}

func (s *Server) Start() error {
	s.logger.Info("http server listening", "addr", s.server.Addr)
	err := s.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"service": "gopick",
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	wsStatus := domain.WSStatus{}
	if s.wsStatus != nil {
		wsStatus = s.wsStatus.Status()
	}

	writeJSON(w, http.StatusOK, statusResponse{
		Service: "gopick",
		Status:  "ready",
		Now:     time.Now().Format(time.RFC3339),
		Routes: []string{
			"GET /health",
			"GET /status",
			"GET /order-context",
			"GET /list-orders",
			"GET /list-orders/{status}",
			"GET /all-orders",
			"GET /all-orders/{date}",
		},
		WS: wsStatus,
	})
}

func (s *Server) handleOrderContext(w http.ResponseWriter, r *http.Request) {
	contextInfo, err := s.orderService.GetOrderContext(r.Context())
	if err != nil {
		s.logger.Error("get order context failed", "error", err)
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, contextInfo)
}

func (s *Server) handleListOrders(w http.ResponseWriter, r *http.Request) {
	status := domain.OrderStatus(strings.TrimSpace(r.PathValue("status")))
	if status == "" {
		status = domain.OrderStatus(strings.TrimSpace(r.URL.Query().Get("status")))
	}
	if status == "" {
		status = domain.OrderStatusConfirm
	}

	if !domain.IsAllowedOrderStatus(status) {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": fmt.Sprintf("unsupported status: %s", status),
		})
		return
	}

	orders, err := s.orderService.ListOrders(r.Context(), status)
	if err != nil {
		s.logger.Error("list orders failed", "error", err, "status", status)
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, orders)
}

func (s *Server) handleAllOrders(w http.ResponseWriter, r *http.Request) {
	date := strings.TrimSpace(r.PathValue("date"))
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	orders, err := s.orderService.ListAllOrders(r.Context(), date)
	if err != nil {
		s.logger.Error("list all orders failed", "error", err, "date", date)
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, orders)
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Info("request completed", "method", r.Method, "path", r.URL.Path, "duration", time.Since(startedAt))
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
