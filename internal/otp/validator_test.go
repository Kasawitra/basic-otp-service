package otp

import (
	"errors"
	"testing"
)

func TestValidateE164Phone(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid Indonesian number", input: "+628123456789", want: "+628123456789"},
		{name: "valid US number with surrounding spaces", input: " +14155552671 ", want: "+14155552671"},
		{name: "valid max 15 digits", input: "+123456789012345", want: "+123456789012345"},
		{name: "empty string", input: "", wantErr: true},
		{name: "missing plus sign", input: "628123456789", wantErr: true},
		{name: "leading zero after plus", input: "+08123456789", wantErr: true},
		{name: "contains spaces inside", input: "+62 812 3456 789", wantErr: true},
		{name: "contains dashes", input: "+1-415-555-2671", wantErr: true},
		{name: "too short", input: "+123456", wantErr: true},
		{name: "too long (16 digits)", input: "+1234567890123456", wantErr: true},
		{name: "contains letters", input: "+62812abc6789", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ValidateE164Phone(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil (value %q)", tc.input, got)
				}
				if !errors.Is(err, ErrInvalidPhone) {
					t.Fatalf("expected ErrInvalidPhone, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestValidateOTPCode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid 6-digit code", input: "123456", want: "123456"},
		{name: "valid 4-digit code with spaces", input: " 0429 ", want: "0429"},
		{name: "valid 10-digit code", input: "1234567890", want: "1234567890"},
		{name: "empty string", input: "", wantErr: true},
		{name: "too short", input: "123", wantErr: true},
		{name: "too long", input: "12345678901", wantErr: true},
		{name: "non-numeric", input: "12ab56", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ValidateOTPCode(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil (value %q)", tc.input, got)
				}
				if !errors.Is(err, ErrInvalidCode) {
					t.Fatalf("expected ErrInvalidCode, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
