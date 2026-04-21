package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/asmisnik/users-notifier/internal/metrics"
	"github.com/asmisnik/users-notifier/internal/telegram"
)

type SendRequest struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

type Handler struct {
	tg     *telegram.Client
	logger *zap.Logger
}

func New(tg *telegram.Client, logger *zap.Logger) *Handler {
	return &Handler{tg: tg, logger: logger}
}

func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.ChatID == 0 {
		http.Error(w, "chat_id is required", http.StatusBadRequest)
		return
	}
	if req.Text == "" {
		http.Error(w, "text is required", http.StatusBadRequest)
		return
	}

	start := time.Now()
	err := h.tg.SendMessage(req.ChatID, req.Text)
	metrics.SendDuration.Observe(time.Since(start).Seconds())

	if err != nil {
		metrics.MessagesFailed.Inc()
		h.logger.Warn("telegram send failed",
			zap.Int64("chat_id", req.ChatID),
			zap.Error(err),
		)
		http.Error(w, "failed to send message", http.StatusBadGateway)
		return
	}

	metrics.MessagesSent.Inc()
	h.logger.Info("message sent",
		zap.Int64("chat_id", req.ChatID),
		zap.Int("text_len", len(req.Text)),
	)
	w.WriteHeader(http.StatusOK)
}