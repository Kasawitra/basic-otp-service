package otp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
)

const maxRequestBodyBytes = 1 << 20 // 1 MB

// OTPService defines the service operations required by the HTTP handlers.
type OTPService interface {
	SendOTP(ctx context.Context, phone string) (*SendResult, error)
	VerifyOTP(ctx context.Context, phone, code string) (*VerifyResult, error)
}

// Handler exposes HTTP endpoints for sending and verifying OTPs.
type Handler struct {
	service OTPService
	logger  *slog.Logger
}

// NewHandler creates a new HTTP Handler for OTP operations.
func NewHandler(service OTPService, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers all OTP and health endpoints on mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.handleHealth)
	mux.HandleFunc("POST /api/v1/otp/send", h.handleSendOTP)
	mux.HandleFunc("POST /api/v1/otp/verify", h.handleVerifyOTP)
}

type sendOTPRequest struct {
	Phone string `json:"phone"`
}

type verifyOTPRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleSendOTP(w http.ResponseWriter, r *http.Request) {
	var req sendOTPRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	result, err := h.service.SendOTP(r.Context(), req.Phone)
	if err != nil {
		h.writeServiceError(w, "send_otp", err)
		return
	}

	h.logger.Info("otp sent",
		slog.String("sid", result.SID),
		slog.String("channel", result.Channel),
		slog.String("status", result.Status),
	)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleVerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req verifyOTPRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	result, err := h.service.VerifyOTP(r.Context(), req.Phone, req.Code)
	if err != nil {
		h.writeServiceError(w, "verify_otp", err)
		return
	}

	h.logger.Info("otp checked",
		slog.String("sid", result.SID),
		slog.String("status", result.Status),
		slog.Bool("valid", result.Valid),
	)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) writeServiceError(w http.ResponseWriter, op string, err error) {
	switch {
	case errors.Is(err, ErrInvalidPhone), errors.Is(err, ErrInvalidCode):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, ErrProviderBadRequest):
		h.logger.Warn("provider rejected request", slog.String("op", op), slog.String("error", err.Error()))
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid phone number or verification request"})
	case errors.Is(err, ErrVerificationNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "verification not found or expired"})
	case errors.Is(err, ErrRateLimited):
		h.logger.Warn("rate limited by Twilio Verify", slog.String("op", op), slog.String("error", err.Error()))
		writeJSON(w, http.StatusTooManyRequests, errorResponse{Error: "too many verification attempts, please try again later"})
	case errors.Is(err, ErrChannelUnavailable):
		h.logger.Error("verification channel unavailable", slog.String("op", op), slog.String("error", err.Error()))
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "verification channel is currently unavailable"})
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		writeJSON(w, http.StatusRequestTimeout, errorResponse{Error: "request canceled or timed out"})
	default:
		h.logger.Error("unexpected verification error", slog.String("op", op), slog.String("error", err.Error()))
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: "failed to process verification request"})
	}
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("request body must not be empty")
		}
		return errors.New("invalid JSON request body")
	}

	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON object")
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
