package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
)

type Handler struct {
	w     io.Writer
	level slog.Leveler
	mu    *sync.Mutex
	attrs []slog.Attr
	group string
}

func NewHandler(w io.Writer, level slog.Leveler) *Handler {
	return &Handler{
		w:     w,
		level: level,
		mu:    &sync.Mutex{},
	}
}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *Handler) Handle(_ context.Context, record slog.Record) error {
	attrs := make([]slog.Attr, 0, len(h.attrs)+record.NumAttrs())
	attrs = append(attrs, h.attrs...)
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})

	line := h.formatRecord(record.Message, attrs)

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.w, line+"\n")
	return err
	}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	cloned := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	cloned = append(cloned, h.attrs...)
	cloned = append(cloned, attrs...)
	return &Handler{w: h.w, level: h.level, mu: h.mu, attrs: cloned, group: h.group}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	group := name
	if h.group != "" {
		group = h.group + "." + name
	}
	return &Handler{w: h.w, level: h.level, mu: h.mu, attrs: h.attrs, group: group}
}

func (h *Handler) formatRecord(message string, attrs []slog.Attr) string {
	values := make(map[string]string, len(attrs))
	for _, attr := range attrs {
		appendAttr(values, h.group, attr)
	}

	switch message {
	case "http server listening":
		addr := values["addr"]
		if addr == "" {
			return "[http] listening"
		}
		return fmt.Sprintf("[http] listening on %s", addr)
	case "maiyatian ws message":
		cmd := values["cmd"]
		msg := values["msg"]
		if cmd == "" {
			return "[ws]"
		}
		if msg == "" {
			return fmt.Sprintf("[ws] %s", cmd)
		}
		return fmt.Sprintf("[ws] %s: %s", cmd, msg)
	case "shutting down":
		return "shutting down"
		}

	parts := []string{message}
	for _, attr := range attrs {
		key, value, ok := renderAttr(h.group, attr)
		if !ok {
			continue
		}
		if key == "error" {
			parts = append(parts, ": "+value)
			continue
		}
		parts = append(parts, fmt.Sprintf(" %s=%s", key, value))
	}
	return strings.Join(parts, "")
}

func appendAttr(values map[string]string, group string, attr slog.Attr) {
	key, value, ok := renderAttr(group, attr)
	if !ok {
		return
	}
	values[key] = value
}

func renderAttr(group string, attr slog.Attr) (string, string, bool) {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return "", "", false
	}

	key := attr.Key
	if group != "" {
		key = group + "." + key
	}

	if attr.Value.Kind() == slog.KindGroup {
		parts := make([]string, 0, len(attr.Value.Group()))
		for _, nested := range attr.Value.Group() {
			nestedKey, nestedValue, ok := renderAttr(key, nested)
			if !ok {
				continue
			}
			parts = append(parts, fmt.Sprintf("%s=%s", nestedKey, nestedValue))
		}
		return key, strings.Join(parts, " "), len(parts) > 0
	}

	return key, attr.Value.String(), true
}
