package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/asmisnik/users-notifier/internal/metrics"
	"github.com/asmisnik/users-notifier/internal/telegram"
)

// maxDocumentSize caps the multipart upload accepted by SendDocument
// (32 MiB, comfortably above what a CSV report is expected to reach).
const maxDocumentSize = 32 << 20

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

// SendDocument accepts a multipart/form-data upload with fields "chat_id",
// optional "caption", and a "document" file part, and forwards it to
// Telegram as a document message.
func (h *Handler) SendDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxDocumentSize)
	if err := r.ParseMultipartForm(maxDocumentSize); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	chatID, err := strconv.ParseInt(r.FormValue("chat_id"), 10, 64)
	if err != nil || chatID == 0 {
		http.Error(w, "chat_id is required", http.StatusBadRequest)
		return
	}
	caption := r.FormValue("caption")

	file, header, err := r.FormFile("document")
	if err != nil {
		http.Error(w, "document file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "reading file", http.StatusBadRequest)
		return
	}

	start := time.Now()
	err = h.tg.SendDocument(chatID, header.Filename, data, caption)
	metrics.SendDuration.Observe(time.Since(start).Seconds())

	if err != nil {
		metrics.MessagesFailed.Inc()
		h.logger.Warn("telegram send document failed",
			zap.Int64("chat_id", chatID),
			zap.Error(err),
		)
		http.Error(w, "failed to send document", http.StatusBadGateway)
		return
	}

	metrics.MessagesSent.Inc()
	h.logger.Info("document sent",
		zap.Int64("chat_id", chatID),
		zap.String("filename", header.Filename),
		zap.Int("size_bytes", len(data)),
	)
	w.WriteHeader(http.StatusOK)
}
