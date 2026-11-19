package otp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/twilio/twilio-go"
	twilioclient "github.com/twilio/twilio-go/client"
	verify "github.com/twilio/twilio-go/rest/verify/v2"
)

const (
	ChannelWhatsApp = "whatsapp"
	ChannelSMS      = "sms"
)

var (
	// ErrRateLimited indicates Twilio Verify rate limits or max attempts were exceeded.
	ErrRateLimited = errors.New("rate limit or max attempts reached")
	// ErrVerificationNotFound indicates no pending verification exists (expired, already used, or never created).
	ErrVerificationNotFound = errors.New("verification not found or expired")
	// ErrChannelUnavailable indicates the requested delivery channel is not configured or disabled in Twilio Verify.
	ErrChannelUnavailable = errors.New("verification channel unavailable")
	// ErrProviderBadRequest indicates Twilio rejected the request parameters (e.g. unsupported phone number).
	ErrProviderBadRequest = errors.New("invalid verification request")
	// ErrProviderFailure indicates an unexpected upstream Twilio error.
	ErrProviderFailure = errors.New("verification provider error")
)

// VerifyClient abstracts the Twilio Verify v2 SDK methods used by Service.
type VerifyClient interface {
	CreateVerification(serviceSid string, params *verify.CreateVerificationParams) (*verify.VerifyV2Verification, error)
	CreateVerificationCheck(serviceSid string, params *verify.CreateVerificationCheckParams) (*verify.VerifyV2VerificationCheck, error)
}

// SendResult contains the outcome of initiating an OTP verification.
type SendResult struct {
	SID     string `json:"sid"`
	To      string `json:"to"`
	Channel string `json:"channel"`
	Status  string `json:"status"`
}

// VerifyResult contains the outcome of checking an OTP code.
type VerifyResult struct {
	SID    string `json:"sid"`
	To     string `json:"to"`
	Status string `json:"status"`
	Valid  bool   `json:"valid"`
}

// Service manages OTP delivery and verification using Twilio Verify v2.
type Service struct {
	client                  VerifyClient
	serviceSID              string
	fallbackToSMSOnAPIError bool
	logger                  *slog.Logger
}

// NewTwilioService creates a Service backed by the official twilio-go REST client.
func NewTwilioService(accountSID, authToken, serviceSID string, fallbackToSMSOnAPIError bool, logger *slog.Logger) *Service {
	restClient := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSID,
		Password: authToken,
	})
	return NewService(restClient.VerifyV2, serviceSID, fallbackToSMSOnAPIError, logger)
}

// NewService creates a Service with the provided VerifyClient.
func NewService(client VerifyClient, serviceSID string, fallbackToSMSOnAPIError bool, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		client:                  client,
		serviceSID:              serviceSID,
		fallbackToSMSOnAPIError: fallbackToSMSOnAPIError,
		logger:                  logger,
	}
}

// SendOTP initiates an OTP verification via WhatsApp.
// Twilio Verify natively falls back to SMS if WhatsApp delivery fails (e.g. recipient not on WhatsApp).
// When fallbackToSMSOnAPIError is enabled, an immediate channel-unavailable API error also retries via SMS.
func (s *Service) SendOTP(ctx context.Context, phone string) (*SendResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	normalizedPhone, err := ValidateE164Phone(phone)
	if err != nil {
		return nil, err
	}

	res, err := s.createVerification(normalizedPhone, ChannelWhatsApp)
	if err != nil {
		mappedErr := mapTwilioError(err)
		if s.fallbackToSMSOnAPIError && errors.Is(mappedErr, ErrChannelUnavailable) {
			s.logger.Warn("whatsapp verification rejected by Twilio API; falling back to SMS",
				slog.String("error", err.Error()),
			)
			res, err = s.createVerification(normalizedPhone, ChannelSMS)
			if err != nil {
				return nil, mapTwilioError(err)
			}
			return res, nil
		}
		return nil, mappedErr
	}

	return res, nil
}

// VerifyOTP checks the submitted OTP code for the given phone number.
func (s *Service) VerifyOTP(ctx context.Context, phone, code string) (*VerifyResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	normalizedPhone, err := ValidateE164Phone(phone)
	if err != nil {
		return nil, err
	}
	normalizedCode, err := ValidateOTPCode(code)
	if err != nil {
		return nil, err
	}

	params := &verify.CreateVerificationCheckParams{}
	params.SetTo(normalizedPhone)
	params.SetCode(normalizedCode)

	resp, err := s.client.CreateVerificationCheck(s.serviceSID, params)
	if err != nil {
		return nil, mapTwilioError(err)
	}

	status := derefString(resp.Status)
	return &VerifyResult{
		SID:    derefString(resp.Sid),
		To:     derefString(resp.To),
		Status: status,
		Valid:  status == "approved",
	}, nil
}

func (s *Service) createVerification(phone, channel string) (*SendResult, error) {
	params := &verify.CreateVerificationParams{}
	params.SetTo(phone)
	params.SetChannel(channel)

	resp, err := s.client.CreateVerification(s.serviceSID, params)
	if err != nil {
		return nil, err
	}

	returnedChannel := derefString(resp.Channel)
	if returnedChannel == "" {
		returnedChannel = channel
	}

	return &SendResult{
		SID:     derefString(resp.Sid),
		To:      derefString(resp.To),
		Channel: returnedChannel,
		Status:  derefString(resp.Status),
	}, nil
}

func mapTwilioError(err error) error {
	var restErr *twilioclient.TwilioRestError
	if !errors.As(err, &restErr) {
		return fmt.Errorf("%w: %v", ErrProviderFailure, err)
	}

	switch restErr.Code {
	case 60203, 60202, 20429:
		return fmt.Errorf("%w: %s (twilio code %d)", ErrRateLimited, restErr.Message, restErr.Code)
	case 20404:
		return fmt.Errorf("%w: %s (twilio code %d)", ErrVerificationNotFound, restErr.Message, restErr.Code)
	case 68008, 60223:
		return fmt.Errorf("%w: %s (twilio code %d)", ErrChannelUnavailable, restErr.Message, restErr.Code)
	case 60200, 60205:
		return fmt.Errorf("%w: %s (twilio code %d)", ErrProviderBadRequest, restErr.Message, restErr.Code)
	}

	switch restErr.Status {
	case http.StatusTooManyRequests:
		return fmt.Errorf("%w: %s (twilio code %d)", ErrRateLimited, restErr.Message, restErr.Code)
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s (twilio code %d)", ErrVerificationNotFound, restErr.Message, restErr.Code)
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s (twilio code %d)", ErrProviderBadRequest, restErr.Message, restErr.Code)
	default:
		return fmt.Errorf("%w: %s (twilio code %d)", ErrProviderFailure, restErr.Message, restErr.Code)
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
