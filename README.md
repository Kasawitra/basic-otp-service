# Twilio OTP Service (Go + Twilio Verify)

A Go REST API for sending and verifying OTPs using the official `twilio-go` SDK and **Twilio Verify API v2**. OTPs are sent via **WhatsApp** first, using Twilio Verify's native **SMS fallback** when WhatsApp delivery is unavailable.

## Prerequisites

- Go 1.24+
- A Twilio account with:
  1. **Account SID** (`AC...`) and **Auth Token**
  2. A **Verify Service** (`VA...`) created under **Twilio Console → Verify → Services**
  3. **WhatsApp** configured on your Verify Service and **SMS** enabled for automatic fallback

## Setup

1. Copy the environment template:
   ```bash
   cp .env.example .env
   ```
2. Fill in your Twilio credentials in `.env`:
   ```env
   PORT=8080
   TWILIO_ACCOUNT_SID=ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
   TWILIO_AUTH_TOKEN=your_auth_token_here
   TWILIO_VERIFY_SERVICE_SID=VAxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
   FALLBACK_TO_SMS_ON_API_ERROR=false
   ```
3. Run the server:
   ```bash
   go run ./cmd/server
   ```

## API Endpoints

### Health Check
```bash
curl http://localhost:8080/health
```

### Send OTP
Phone numbers must be in strict **E.164** format (e.g. `+628123456789`).
```bash
curl -X POST http://localhost:8080/api/v1/otp/send \
  -H "Content-Type: application/json" \
  -d '{"phone": "+628123456789"}'
```

Response (`200 OK`):
```json
{
  "sid": "VExxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "to": "+628123456789",
  "channel": "whatsapp",
  "status": "pending"
}
```

### Verify OTP
```bash
curl -X POST http://localhost:8080/api/v1/otp/verify \
  -H "Content-Type: application/json" \
  -d '{"phone": "+628123456789", "code": "123456"}'
```

Response (`200 OK`):
```json
{
  "sid": "VExxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "to": "+628123456789",
  "status": "approved",
  "valid": true
}
```
