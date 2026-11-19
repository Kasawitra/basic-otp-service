package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds the runtime configuration for the OTP service.
type Config struct {
	Port                    string
	TwilioAccountSID        string
	TwilioAuthToken         string
	TwilioVerifyServiceSID  string
	FallbackToSMSOnAPIError bool
}

// Load reads configuration from an optional .env file and environment variables.
func Load() (*Config, error) {
	// Load .env if present; ignore if the file does not exist (e.g. in production).
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load .env file: %w", err)
	}

	cfg := &Config{
		Port:                   strings.TrimSpace(os.Getenv("PORT")),
		TwilioAccountSID:       strings.TrimSpace(os.Getenv("TWILIO_ACCOUNT_SID")),
		TwilioAuthToken:        strings.TrimSpace(os.Getenv("TWILIO_AUTH_TOKEN")),
		TwilioVerifyServiceSID: strings.TrimSpace(os.Getenv("TWILIO_VERIFY_SERVICE_SID")),
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	if rawFallback := strings.TrimSpace(os.Getenv("FALLBACK_TO_SMS_ON_API_ERROR")); rawFallback != "" {
		val, err := strconv.ParseBool(rawFallback)
		if err != nil {
			return nil, fmt.Errorf("invalid FALLBACK_TO_SMS_ON_API_ERROR value %q: %w", rawFallback, err)
		}
		cfg.FallbackToSMSOnAPIError = val
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	var missing []string
	if c.TwilioAccountSID == "" {
		missing = append(missing, "TWILIO_ACCOUNT_SID")
	}
	if c.TwilioAuthToken == "" {
		missing = append(missing, "TWILIO_AUTH_TOKEN")
	}
	if c.TwilioVerifyServiceSID == "" {
		missing = append(missing, "TWILIO_VERIFY_SERVICE_SID")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	if !strings.HasPrefix(c.TwilioAccountSID, "AC") {
		return errors.New("TWILIO_ACCOUNT_SID must start with 'AC'")
	}
	if !strings.HasPrefix(c.TwilioVerifyServiceSID, "VA") {
		return errors.New("TWILIO_VERIFY_SERVICE_SID must start with 'VA'")
	}

	return nil
}
