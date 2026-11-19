package otp

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// ErrInvalidPhone is returned when a phone number is not valid E.164 format.
	ErrInvalidPhone = errors.New("invalid phone number")
	// ErrInvalidCode is returned when an OTP code does not match the expected format.
	ErrInvalidCode = errors.New("invalid OTP code")

	// e164Regex matches strict E.164 phone numbers: '+' followed by 7 to 15 digits,
	// with the first digit between 1 and 9 (no leading zero after '+').
	e164Regex = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)

	// otpCodeRegex matches numeric OTP codes between 4 and 10 digits (Twilio Verify range).
	otpCodeRegex = regexp.MustCompile(`^\d{4,10}$`)
)

// ValidateE164Phone trims whitespace and verifies that phone is in strict E.164 format
// (for example, "+628123456789" or "+14155552671").
func ValidateE164Phone(phone string) (string, error) {
	cleaned := strings.TrimSpace(phone)
	if cleaned == "" {
		return "", fmt.Errorf("%w: phone number is required", ErrInvalidPhone)
	}
	if !e164Regex.MatchString(cleaned) {
		return "", fmt.Errorf("%w: must be in E.164 format (e.g. +628123456789)", ErrInvalidPhone)
	}
	return cleaned, nil
}

// ValidateOTPCode trims whitespace and verifies that code is a 4 to 10 digit numeric string.
func ValidateOTPCode(code string) (string, error) {
	cleaned := strings.TrimSpace(code)
	if cleaned == "" {
		return "", fmt.Errorf("%w: code is required", ErrInvalidCode)
	}
	if !otpCodeRegex.MatchString(cleaned) {
		return "", fmt.Errorf("%w: must be 4 to 10 numeric digits", ErrInvalidCode)
	}
	return cleaned, nil
}
