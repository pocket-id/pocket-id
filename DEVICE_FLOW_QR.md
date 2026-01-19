# OIDC Device Flow with QR Code Authentication

## Overview

Pocket ID now supports the **OIDC Device Authorization Grant (RFC 8628)** with QR code support, enabling passwordless authentication for remote devices such as:

- Smart TVs
- IoT devices
- Kiosks
- Command-line applications
- Any device with limited input capabilities

This implementation allows users to authenticate on a trusted device (smartphone) by scanning a QR code, eliminating the need to type passwords on devices with difficult input methods.

## How It Works

The device flow with QR code follows these steps:

```
┌─────────────┐                                    ┌──────────────┐
│   Remote    │                                    │   Identity   │
│   Device    │                                    │   Provider   │
│  (TV/IoT)   │                                    │ (Pocket ID)  │
└──────┬──────┘                                    └──────┬───────┘
       │                                                  │
       │ 1. Request Device Authorization                 │
       │ POST /api/oidc/device/authorize                 │
       │────────────────────────────────────────────────>│
       │                                                  │
       │ 2. Receive device_code, user_code, QR URI       │
       │<────────────────────────────────────────────────│
       │                                                  │
       │ 3. Display QR Code                              │
       │                                                  │
       │                           ┌──────────────┐      │
       │                           │ Smartphone   │      │
       │                           └──────┬───────┘      │
       │ 4. User scans QR code            │              │
       │<─────────────────────────────────┤              │
       │                                  │              │
       │                                  │ 5. Authenticate
       │                                  │ (Passkey)    │
       │                                  │─────────────>│
       │                                  │              │
       │                                  │ 6. Approve   │
       │                                  │    Device    │
       │                                  │─────────────>│
       │                                  │              │
       │ 7. Poll for tokens (every 5s)                   │
       │────────────────────────────────────────────────>│
       │                                                  │
       │ 8. Receive access_token, id_token               │
       │<────────────────────────────────────────────────│
       │                                                  │
```

## API Endpoints

### 1. Initiate Device Authorization

**Endpoint:** `POST /api/oidc/device/authorize`

**Request:**
```http
POST /api/oidc/device/authorize HTTP/1.1
Content-Type: application/x-www-form-urlencoded

client_id=your-client-id
&client_secret=your-client-secret
&scope=openid profile email
```

**Response:**
```json
{
  "device_code": "8c7e4a9b...",
  "user_code": "ABCD-1234",
  "verification_uri": "https://your-pocket-id.com/device",
  "verification_uri_complete": "https://your-pocket-id.com/device?code=ABCD-1234",
  "qr_code_uri": "https://your-pocket-id.com/api/oidc/device/qrcode?code=ABCD-1234",
  "expires_in": 900,
  "interval": 5
}
```

**Field Descriptions:**
- `device_code`: Secret code used by the device to poll for tokens
- `user_code`: Short code displayed to the user (8 characters)
- `verification_uri`: URL where the user should authenticate
- `verification_uri_complete`: URL with the user code pre-filled
- `qr_code_uri`: URL to fetch the QR code image (PNG format)
- `expires_in`: Time in seconds before the codes expire (default: 900)
- `interval`: Minimum seconds between polling requests (default: 5)

### 2. Get QR Code Image

**Endpoint:** `GET /api/oidc/device/qrcode`

**Request:**
```http
GET /api/oidc/device/qrcode?code=ABCD-1234 HTTP/1.1
```

**Response:**
- Content-Type: `image/png`
- Returns a 256x256 PNG image of the QR code

### 3. Poll for Tokens

**Endpoint:** `POST /api/oidc/token`

**Request:**
```http
POST /api/oidc/token HTTP/1.1
Content-Type: application/x-www-form-urlencoded

grant_type=urn:ietf:params:oauth:grant-type:device_code
&device_code=8c7e4a9b...
&client_id=your-client-id
&client_secret=your-client-secret
```

**Responses:**

**Still Pending:**
```json
HTTP/1.1 400 Bad Request
{
  "error": "authorization_pending"
}
```

**Success:**
```json
HTTP/1.1 200 OK
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "id_token": "eyJhbGciOiJSUzI1NiIs...",
  "refresh_token": "8xLOxBtZp8..."
}
```

**Other Error Responses:**
- `slow_down`: Client should increase polling interval
- `expired_token`: Device code has expired
- `access_denied`: User denied the authorization

## Implementation Examples

### JavaScript/TypeScript Example

```javascript
class DeviceFlowAuth {
  constructor(config) {
    this.baseUrl = config.baseUrl;
    this.clientId = config.clientId;
    this.clientSecret = config.clientSecret;
    this.scope = config.scope || 'openid profile email';
  }

  async startFlow() {
    // Step 1: Request device authorization
    const response = await fetch(`${this.baseUrl}/api/oidc/device/authorize`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams({
        client_id: this.clientId,
        client_secret: this.clientSecret,
        scope: this.scope
      })
    });

    const data = await response.json();
    
    // Step 2: Display QR code to user
    this.displayQRCode(data.qr_code_uri, data.user_code);
    
    // Step 3: Start polling
    return this.pollForTokens(data.device_code, data.interval);
  }

  displayQRCode(qrCodeUri, userCode) {
    // Display the QR code image
    const img = document.createElement('img');
    img.src = qrCodeUri;
    document.body.appendChild(img);
    
    // Also display the user code as a fallback
    console.log(`User Code: ${userCode}`);
  }

  async pollForTokens(deviceCode, interval) {
    return new Promise((resolve, reject) => {
      const poll = setInterval(async () => {
        try {
          const response = await fetch(`${this.baseUrl}/api/oidc/token`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: new URLSearchParams({
              grant_type: 'urn:ietf:params:oauth:grant-type:device_code',
              device_code: deviceCode,
              client_id: this.clientId,
              client_secret: this.clientSecret
            })
          });

          if (response.ok) {
            clearInterval(poll);
            const tokens = await response.json();
            resolve(tokens);
          } else {
            const error = await response.json();
            if (error.error === 'authorization_pending' || error.error === 'slow_down') {
              // Continue polling
              return;
            }
            clearInterval(poll);
            reject(new Error(error.error));
          }
        } catch (err) {
          clearInterval(poll);
          reject(err);
        }
      }, interval * 1000);
    });
  }
}

// Usage
const auth = new DeviceFlowAuth({
  baseUrl: 'https://your-pocket-id.com',
  clientId: 'your-client-id',
  clientSecret: 'your-client-secret',
  scope: 'openid profile email'
});

auth.startFlow()
  .then(tokens => {
    console.log('Access Token:', tokens.access_token);
    console.log('ID Token:', tokens.id_token);
  })
  .catch(err => console.error('Error:', err));
```

### Python Example

```python
import requests
import time
import qrcode
from io import BytesIO
from PIL import Image

class DeviceFlowAuth:
    def __init__(self, base_url, client_id, client_secret, scope='openid profile email'):
        self.base_url = base_url
        self.client_id = client_id
        self.client_secret = client_secret
        self.scope = scope

    def start_flow(self):
        # Step 1: Request device authorization
        response = requests.post(
            f'{self.base_url}/api/oidc/device/authorize',
            data={
                'client_id': self.client_id,
                'client_secret': self.client_secret,
                'scope': self.scope
            }
        )
        response.raise_for_status()
        data = response.json()
        
        # Step 2: Display QR code
        self.display_qr_code(data['verification_uri_complete'], data['user_code'])
        
        # Step 3: Poll for tokens
        return self.poll_for_tokens(data['device_code'], data['interval'])

    def display_qr_code(self, url, user_code):
        # Generate and display QR code
        qr = qrcode.QRCode(version=1, box_size=10, border=5)
        qr.add_data(url)
        qr.make(fit=True)
        img = qr.make_image(fill_color="black", back_color="white")
        img.show()
        
        print(f"\nUser Code: {user_code}")
        print(f"Or visit: {url}\n")

    def poll_for_tokens(self, device_code, interval):
        while True:
            time.sleep(interval)
            
            response = requests.post(
                f'{self.base_url}/api/oidc/token',
                data={
                    'grant_type': 'urn:ietf:params:oauth:grant-type:device_code',
                    'device_code': device_code,
                    'client_id': self.client_id,
                    'client_secret': self.client_secret
                }
            )
            
            if response.status_code == 200:
                return response.json()
            
            error_data = response.json()
            error = error_data.get('error', 'unknown')
            
            if error in ['authorization_pending', 'slow_down']:
                continue
            else:
                raise Exception(f'Authorization failed: {error}')

# Usage
auth = DeviceFlowAuth(
    base_url='https://your-pocket-id.com',
    client_id='your-client-id',
    client_secret='your-client-secret'
)

try:
    tokens = auth.start_flow()
    print('Access Token:', tokens['access_token'])
    print('ID Token:', tokens['id_token'])
except Exception as e:
    print(f'Error: {e}')
```

### Go Example

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
	
	"github.com/skip2/go-qrcode"
)

type DeviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	QRCodeURI               string `json:"qr_code_uri"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
}

func startDeviceFlow(baseURL, clientID, clientSecret, scope string) (*TokenResponse, error) {
	// Step 1: Request device authorization
	data := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"scope":         {scope},
	}
	
	resp, err := http.PostForm(baseURL+"/api/oidc/device/authorize", data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var authResp DeviceAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}
	
	// Step 2: Display QR code
	displayQRCode(authResp.VerificationURIComplete, authResp.UserCode)
	
	// Step 3: Poll for tokens
	return pollForTokens(baseURL, clientID, clientSecret, authResp.DeviceCode, authResp.Interval)
}

func displayQRCode(verificationURL, userCode string) {
	// Generate QR code
	qr, _ := qrcode.New(verificationURL, qrcode.Medium)
	fmt.Println(qr.ToSmallString(false))
	fmt.Printf("\nUser Code: %s\n", userCode)
	fmt.Printf("Or visit: %s\n\n", verificationURL)
}

func pollForTokens(baseURL, clientID, clientSecret, deviceCode string, interval int) (*TokenResponse, error) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		data := url.Values{
			"grant_type":    {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code":   {deviceCode},
			"client_id":     {clientID},
			"client_secret": {clientSecret},
		}
		
		resp, err := http.PostForm(baseURL+"/api/oidc/token", data)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		
		if resp.StatusCode == http.StatusOK {
			var tokens TokenResponse
			if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
				return nil, err
			}
			return &tokens, nil
		}
		
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		
		if errResp["error"] == "authorization_pending" || errResp["error"] == "slow_down" {
			continue
		}
		
		return nil, fmt.Errorf("authorization failed: %s", errResp["error"])
	}
	
	return nil, fmt.Errorf("polling timeout")
}

func main() {
	tokens, err := startDeviceFlow(
		"https://your-pocket-id.com",
		"your-client-id",
		"your-client-secret",
		"openid profile email",
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Access Token: %s\n", tokens.AccessToken)
	fmt.Printf("ID Token: %s\n", tokens.IDToken)
}
```

## Security Considerations

1. **Device Code Protection**: The `device_code` should be kept secret on the device. Never expose it in URLs or logs.

2. **Client Secrets**: For confidential clients, the `client_secret` should be securely stored and never exposed to end users.

3. **Rate Limiting**: Respect the `interval` value to avoid overwhelming the server with requests.

4. **Expiration**: Device codes expire after 15 minutes (900 seconds). Handle expiration gracefully.

5. **User Verification**: Users must authenticate with passkeys on their mobile device, providing strong security through biometrics or hardware keys.

## Benefits

- **Passwordless**: No need to type passwords on devices with difficult input
- **Secure**: Authentication happens on a trusted device (smartphone)
- **User Presence**: Requires active user confirmation through biometrics/PIN
- **Standards-Based**: Implements RFC 8628 (OAuth 2.0 Device Authorization Grant)
- **Cross-Device**: Seamlessly works across devices without shared cookies

## Demo

Visit `/device-demo` on your Pocket ID instance to see a live demonstration of the device flow with QR codes.

## References

- [RFC 8628: OAuth 2.0 Device Authorization Grant](https://datatracker.ietf.org/doc/html/rfc8628)
- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)
- [WebAuthn / Passkeys](https://webauthn.io/)
