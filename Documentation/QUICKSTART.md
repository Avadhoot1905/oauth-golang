# Quick Start Guide

## 🚀 Getting Started in 5 Minutes

### Step 1: Install Dependencies

```bash
go mod download
```

### Step 2: Configure Environment

Update your `.env` file with your Google OAuth credentials:

```env
GOOGLE_CLIENT_ID="your-google-client-id-here"
GOOGLE_CLIENT_SECRET="your-google-client-secret-here"
GOOGLE_REDIRECT_URL="http://localhost:8080/callback"
PORT="8080"
JWT_SECRET="your-super-secret-jwt-key-change-this-in-production"
DATABASE_URL="postgresql://user:password@host:port/database"
```

### Step 3: Run the Server

```bash
go run cmd/server/main.go
```

Or build and run:

```bash
go build -o bin/oauth-server ./cmd/server/main.go
./bin/oauth-server
```

The server will start on `http://localhost:8080`

**Note:** The database tables (`users`, `oauth_clients`, `refresh_tokens`) will be automatically created on first run using GORM auto-migration. A development OAuth client (`demo-frontend`) is automatically seeded for testing!

### Step 4: Use the Auto-Seeded Development Client

The server automatically creates a development OAuth client on startup:

```
Client ID: demo-frontend
Client Secret: dev-secret
Client Name: Demo Frontend App
Client Type: public
Redirect URIs: ["http://localhost:3000/callback"]
Grant Types: ["authorization_code", "refresh_token"]
Scope: openid profile email
```

You can use this client immediately for testing, or create additional test clients:

#### Option A: Use Auto-Seeded Client (Recommended)
Just use `client_id=demo-frontend` in your requests!

#### Option B: Register Additional Test Clients

Create additional test clients using GORM. You can either:

**Option 1: Add this code to a setup script or run it once:**

```go
package main

import (
    "log"
    "oauth-golang/internal/storage"
    "github.com/lib/pq"
)

func main() {
    // Initialize database (also seeds demo-frontend)
    storage.InitDB("postgresql://user:password@localhost:5432/database")
    
    // Create additional test client
    testClient := storage.OAuthClient{
        ClientID:     "test-client",
        ClientSecret: "test-secret",
        ClientName:   "Test Application",
        ClientType:   "confidential",
        RedirectURIs: pq.StringArray{"http://localhost:3000/callback", "http://localhost:8080/callback"},
        GrantTypes:   pq.StringArray{"authorization_code", "refresh_token"},
        Scope:        "openid email profile",
    }
    
    result := storage.DB.Create(&testClient)
    if result.Error != nil {
        log.Fatalf("Failed to create test client: %v", result.Error)
    }
    
    log.Println("Test client created successfully!")
}
```

**Option 2: Use GORM CLI or connect directly via SQL (if preferred):**

```sql
INSERT INTO oauth_clients (client_id, client_secret, client_name, client_type, redirect_uris, grant_types, scope, created_at, updated_at)
VALUES ('test-client', 'test-secret', 'Test Application', 'confidential', 
        ARRAY['http://localhost:3000/callback', 'http://localhost:8080/callback'], 
        ARRAY['authorization_code', 'refresh_token'], 'openid email profile', NOW(), NOW());
```

### Step 5: Test the OAuth Flow

#### 5.1 Initiate Authorization

Open in your browser (using the auto-seeded demo-frontend client):
```
http://localhost:8080/authorize?client_id=demo-frontend&redirect_uri=http://localhost:3000/callback&response_type=code&state=random-state-123
```

You'll be redirected to Google login. After authentication, you'll receive an authorization code.

#### 5.2 Exchange Code for Tokens

```bash
curl -X POST http://localhost:8080/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code" \
  -d "code=YOUR_AUTHORIZATION_CODE" \
  -d "redirect_uri=http://localhost:3000/callback" \
  -d "client_id=demo-frontend" \
  -d "client_secret=dev-secret"
```

Response:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "id_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

#### 5.3 Get User Information

```bash
curl http://localhost:8080/userinfo \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

Response:
```json
{
  "sub": "user-id-12345",
  "email": "user@example.com",
  "email_verified": true,
  "name": "John Doe",
  "given_name": "John",
  "family_name": "Doe",
  "picture": "https://..."
}
```

#### 5.4 Introspect Token

```bash
curl -X POST http://localhost:8080/introspect \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "token=YOUR_ACCESS_TOKEN"
```

Response:
```json
{
  "active": true,
  "scope": "openid email profile",
  "client_id": "oauth-service",
  "username": "user@example.com",
  "token_type": "Bearer",
  "exp": 1733500000,
  "iat": 1733496400,
  "sub": "user-id-12345"
}
```

#### 5.5 Refresh Tokens

```bash
curl -X POST http://localhost:8080/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=refresh_token" \
  -d "refresh_token=YOUR_REFRESH_TOKEN" \
  -d "client_id=test-client" \
  -d "client_secret=test-secret"
```

## 📁 Project Structure Overview

```
oauth-golang/
├── cmd/server/main.go              # Entry point
├── internal/
│   ├── config/config.go            # Config loader
│   ├── http/
│   │   ├── router.go               # HTTP routes
│   │   └── handlers/               # API endpoints
│   ├── oauth/                      # OAuth logic
│   ├── security/                   # JWT & crypto
│   ├── storage/                    # Database layer
│   ├── user/                       # User auth
│   └── models/                     # Shared data models
├── pkg/utils/                      # Utility functions
├── .env                            # Configuration
└── go.mod                          # Dependencies
```

## 🔍 Key Files Explained

### API Input Layer (HTTP Handlers)
- **`handlers/authorize.go`** - Validates OAuth requests, redirects to Google
- **`handlers/token.go`** - Exchanges codes for JWT tokens
- **`handlers/userinfo.go`** - Returns user info from token
- **`handlers/introspect.go`** - Validates tokens for microservices

### Business Logic Layer
- **`oauth/authcode.go`** - Manages authorization codes
- **`oauth/token_service.go`** - Generates and refreshes tokens
- **`oauth/client_registry.go`** - Validates OAuth clients
- **`oauth/pkce.go`** - PKCE security validation
- **`user/auth.go`** - User authentication logic

### Security Layer
- **`security/jwt.go`** - JWT signing and verification
- **`security/hasher.go`** - Password hashing
- **`security/keys.go`** - RSA key management

### Data Access Layer
- **`storage/user_repo.go`** - User CRUD operations (includes User model with GORM tags)
- **`storage/client_repo.go`** - Client CRUD operations (includes OAuthClient model with GORM tags)
- **`storage/token_repo.go`** - Token storage & revocation (includes RefreshToken model with GORM tags)
- **`storage/db.go`** - Database setup with GORM auto-migration

## 🎯 Common Use Cases

### Use Case 1: Single Page Application (SPA)

SPAs should use PKCE for security:

```javascript
// 1. Generate code verifier and challenge
const codeVerifier = generateRandomString(64);
const codeChallenge = await sha256(codeVerifier);

// 2. Redirect to authorization endpoint
window.location.href = `http://localhost:8080/authorize?` +
  `client_id=spa-client&` +
  `redirect_uri=http://localhost:3000/callback&` +
  `response_type=code&` +
  `state=${randomState}&` +
  `code_challenge=${codeChallenge}&` +
  `code_challenge_method=S256`;

// 3. After redirect, exchange code
const response = await fetch('http://localhost:8080/token', {
  method: 'POST',
  headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
  body: new URLSearchParams({
    grant_type: 'authorization_code',
    code: authCode,
    redirect_uri: 'http://localhost:3000/callback',
    client_id: 'spa-client',
    code_verifier: codeVerifier
  })
});
```

### Use Case 2: Backend Microservice

Validate tokens from other services:

```go
// Validate access token
resp, _ := http.Post("http://localhost:8080/introspect",
    "application/x-www-form-urlencoded",
    strings.NewReader("token=" + accessToken))

var result map[string]interface{}
json.NewDecoder(resp.Body).Decode(&result)

if result["active"] == true {
    // Token is valid, proceed
    userID := result["sub"].(string)
}
```

### Use Case 3: Mobile Application

Mobile apps should also use PKCE:

```swift
// 1. Generate PKCE parameters
let codeVerifier = generateRandomString(length: 64)
let codeChallenge = sha256(codeVerifier)

// 2. Open authorization URL
let authURL = URL(string: "http://localhost:8080/authorize?" +
    "client_id=mobile-app&" +
    "redirect_uri=myapp://callback&" +
    "response_type=code&" +
    "state=\(state)&" +
    "code_challenge=\(codeChallenge)&" +
    "code_challenge_method=S256")!

// 3. Exchange code for tokens
let params = [
    "grant_type": "authorization_code",
    "code": authCode,
    "redirect_uri": "myapp://callback",
    "client_id": "mobile-app",
    "code_verifier": codeVerifier
]
```

## 🐛 Troubleshooting

### Issue: "Failed to connect to database"
- Check DATABASE_URL is correct (format: `postgresql://user:password@host:port/database`)
- Ensure PostgreSQL is running
- Verify network connectivity
- Check that the database exists (GORM will create tables but not the database itself)

### Issue: "Invalid client_id"
- Ensure client is registered in oauth_clients table
- Check client_id matches exactly

### Issue: "Invalid redirect_uri"
- Verify redirect_uri is registered in oauth_clients.redirect_uris
- Must match exactly (including trailing slash)

### Issue: "Token expired"
- Access tokens expire after 1 hour
- Use refresh token to get new access token

### Issue: "PKCE validation failed"
- Ensure code_verifier matches code_challenge
- Check code_challenge_method is S256 or plain

## 📊 Monitoring

### Health Check

```bash
curl http://localhost:8080/health
```

### Database Queries

Check active sessions:
```sql
SELECT COUNT(*) FROM refresh_tokens WHERE expires_at > NOW() AND revoked = false;
```

Check user registrations:
```sql
SELECT COUNT(*), DATE(created_at) FROM users GROUP BY DATE(created_at);
```

## 🔐 Security Best Practices

1. **Always use HTTPS in production**
2. **Rotate JWT secrets regularly**
3. **Set strong client secrets**
4. **Implement rate limiting**
5. **Monitor for suspicious activity**
6. **Use short-lived access tokens**
7. **Implement token revocation**
8. **Validate all redirect URIs**
9. **Use PKCE for public clients**
10. **Keep dependencies updated**

## 🎓 Next Steps

1. ✅ Complete basic OAuth flow
2. ⬜ Add email verification
3. ⬜ Implement password reset
4. ⬜ Add 2FA support
5. ⬜ Create admin dashboard
6. ⬜ Add API rate limiting
7. ⬜ Implement logging
8. ⬜ Set up monitoring
9. ⬜ Deploy to production
10. ⬜ Performance testing

## 📚 Additional Resources

- [OAuth 2.0 Spec](https://oauth.net/2/)
- [OpenID Connect](https://openid.net/connect/)
- [PKCE RFC](https://tools.ietf.org/html/rfc7636)
- [JWT Best Practices](https://tools.ietf.org/html/rfc8725)
- [Google OAuth Guide](https://developers.google.com/identity/protocols/oauth2)

---

**Happy Coding! 🚀**
