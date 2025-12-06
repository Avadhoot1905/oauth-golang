# OAuth 2.0 Microservice with Google Authentication

A production-ready OAuth 2.0 authorization server built with Go, implementing Google OAuth integration, JWT token generation, PKCE support, and PostgreSQL persistence. For architectural, implementation or startup details, view documentation directory

## 🚀 Features

- ✅ **Google OAuth 2.0 Integration** - Authorization Code Flow with PKCE
- ✅ **JWT Token Generation** - Access tokens, refresh tokens, and ID tokens
- ✅ **PostgreSQL Database** - User, client, and token persistence
- ✅ **Token Introspection** - Validate tokens for other microservices
- ✅ **PKCE Support** - Enhanced security for public clients (SPAs, mobile apps)
- ✅ **Refresh Token Rotation** - Secure token refresh mechanism
- ✅ **Token Revocation** - Blacklist compromised tokens
- ✅ **RESTful API** - Clean HTTP endpoints following OAuth 2.0 spec
- ✅ **Modular Architecture** - Clean separation of concerns

## 📁 Project Structure

```
oauth-golang/
├── cmd/
│   └── server/
│       └── main.go                    # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go                  # Configuration loader (.env)
│   ├── http/
│   │   ├── router.go                  # HTTP router & middleware
│   │   └── handlers/
│   │       ├── authorize.go           # OAuth authorization endpoint
│   │       ├── token.go               # Token exchange endpoint
│   │       ├── userinfo.go            # User info endpoint
│   │       └── introspect.go          # Token introspection endpoint
│   ├── oauth/
│   │   ├── authcode.go                # Authorization code management
│   │   ├── token_service.go           # Token generation & refresh
│   │   ├── client_registry.go         # OAuth client management
│   │   └── pkce.go                    # PKCE validation
│   ├── security/
│   │   ├── jwt.go                     # JWT signing & verification
│   │   ├── hasher.go                  # Password hashing utilities
│   │   └── keys.go                    # RSA key management
│   ├── storage/
│   │   ├── db.go                      # Database initialization & migrations
│   │   ├── user_repo.go               # User repository
│   │   ├── client_repo.go             # OAuth client repository (GORM-based Storage)
│   │   └── token_repo.go              # Token repository
│   └── user/
│       └── auth.go                    # User authentication logic
├── pkg/
│   └── utils/
│       └── random.go                  # Random string generation
├── .env                               # Environment configuration
├── go.mod                             # Go module dependencies
└── README.md

```

## 🛠️ Prerequisites

- Go 1.21 or higher
- PostgreSQL 14+ or Neon database
- Google Cloud Console project with OAuth 2.0 credentials

## 📦 Installation

### 1. Clone the repository

```bash
git clone <repository-url>
cd oauth-golang
```

### 2. Install dependencies

```bash
go mod download
```

### 3. Set up Google OAuth credentials

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing one
3. Enable Google+ API
4. Create OAuth 2.0 credentials (Web application)
5. Add authorized redirect URI: `http://localhost:8080/callback`
6. Copy Client ID and Client Secret

### 4. Configure environment variables

Create or update `.env` file:

```env
# Google OAuth Configuration
GOOGLE_CLIENT_ID="your-google-client-id"
GOOGLE_CLIENT_SECRET="your-google-client-secret"
GOOGLE_REDIRECT_URL="http://localhost:8080/callback"

# Server Configuration
PORT="8080"

# JWT Configuration
JWT_SECRET="your-super-secret-jwt-key-change-this-in-production"

# Database Configuration (PostgreSQL/Neon)
DATABASE_URL="postgresql://user:password@host:port/database?sslmode=require"
```

### 5. Run database migrations

Migrations run automatically on server start using GORM, creating these tables:
- `users` - User accounts
- `oauth_clients` - OAuth client applications
- `refresh_tokens` - Refresh token storage
- `revoked_tokens` - Token blacklist

**Auto-Seeded Development Client:**
A development OAuth client is automatically created on startup:
- **Client ID:** `demo-frontend`
- **Client Secret:** `dev-secret`
- **Redirect URI:** `http://localhost:3000/callback`
- **Grant Types:** `authorization_code`, `refresh_token`

You can start testing immediately with this client!

### 6. Start the server

```bash
go run cmd/server/main.go
```

Server will start on `http://localhost:8080`

## 🔐 API Endpoints

### 1. **Authorization Endpoint** - `/authorize`

Initiates OAuth 2.0 authorization flow, redirects user to Google login.

**Method:** `GET`

**Query Parameters:**
- `client_id` (required) - OAuth client identifier
- `redirect_uri` (required) - Callback URL after authorization
- `response_type` (required) - Must be `code`
- `state` (optional) - CSRF protection token
- `code_challenge` (optional) - PKCE challenge
- `code_challenge_method` (optional) - `S256` or `plain`
- `scope` (optional) - Requested scopes (default: `openid email profile`)

**Example:**
```bash
# Using the auto-seeded demo-frontend client
curl "http://localhost:8080/authorize?client_id=demo-frontend&redirect_uri=http://localhost:3000/callback&response_type=code&state=random-state&code_challenge=CHALLENGE&code_challenge_method=S256"
```

**Flow:**
1. API INPUT: Client sends authorization request
2. DB INTERACTION: Validate client_id via `client_repo`
3. GOOGLE OAUTH: Redirect to Google login
4. GOOGLE OAUTH: Receive authorization code from Google
5. OUTPUT TO DB: Store authorization code via `authcode_service`
6. Redirect back to client with authorization code

---

### 2. **Token Endpoint** - `/token`

Exchanges authorization code for access token and refresh token.

**Method:** `POST`

**Content-Type:** `application/x-www-form-urlencoded` or `application/json`

**Parameters:**

**For authorization_code grant:**
- `grant_type` (required) - `authorization_code`
- `code` (required) - Authorization code from `/authorize`
- `redirect_uri` (required) - Must match original redirect URI
- `client_id` (required) - OAuth client identifier
- `client_secret` (optional) - Required for confidential clients
- `code_verifier` (optional) - Required if PKCE was used

**For refresh_token grant:**
- `grant_type` (required) - `refresh_token`
- `refresh_token` (required) - Valid refresh token
- `client_id` (required) - OAuth client identifier
- `client_secret` (optional) - Required for confidential clients

**Example:**
```bash
# Using the auto-seeded demo-frontend client
curl -X POST http://localhost:8080/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code" \
  -d "code=AUTHORIZATION_CODE" \
  -d "redirect_uri=http://localhost:3000/callback" \
  -d "client_id=demo-frontend" \
  -d "client_secret=dev-secret" \
  -d "code_verifier=VERIFIER"
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "id_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Flow:**
1. API INPUT: Client sends token request
2. DB INTERACTION: Validate authorization code via `authcode_service`
3. DB INTERACTION: Create or update user via `user_repo`
4. OUTPUT TO DB: Store refresh token via `token_repo`
5. API OUTPUT: Return JWT tokens

---

### 3. **User Info Endpoint** - `/userinfo`

Returns user information from access token (OpenID Connect UserInfo endpoint).

**Method:** `GET` or `POST`

**Headers:**
- `Authorization: Bearer <access_token>`

**Example:**
```bash
curl http://localhost:8080/userinfo \
  -H "Authorization: Bearer ACCESS_TOKEN"
```

**Response:**
```json
{
  "sub": "user-id-12345",
  "email": "user@example.com",
  "email_verified": true,
  "name": "John Doe",
  "given_name": "John",
  "family_name": "Doe",
  "picture": "https://lh3.googleusercontent.com/..."
}
```

**Flow:**
1. API INPUT: Extract Bearer token from Authorization header
2. Verify JWT signature and expiration
3. DB INTERACTION: Retrieve user info via `user_repo`
4. API OUTPUT: Return user information

---

### 4. **Token Introspection Endpoint** - `/introspect`

Validates tokens for other microservices (RFC 7662).

**Method:** `POST`

**Content-Type:** `application/x-www-form-urlencoded` or `application/json`

**Parameters:**
- `token` (required) - Token to introspect
- `token_type_hint` (optional) - `access_token` or `refresh_token`

**Example:**
```bash
curl -X POST http://localhost:8080/introspect \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "token=ACCESS_TOKEN"
```

**Response (Active Token):**
```json
{
  "active": true,
  "scope": "openid email profile",
  "client_id": "oauth-service",
  "username": "user@example.com",
  "token_type": "Bearer",
  "exp": 1733500000,
  "iat": 1733496400,
  "sub": "user-id-12345",
  "aud": "oauth-service",
  "iss": "oauth-service",
  "jti": "token-unique-id"
}
```

**Response (Inactive Token):**
```json
{
  "active": false
}
```

**Flow:**
1. API INPUT: Token to validate
2. Verify JWT signature and expiration
3. DB INTERACTION: Check if token is revoked via `token_repo`
4. API OUTPUT: Return token status and metadata

---

### 5. **Health Check** - `/health`

Simple health check endpoint.

**Method:** `GET`

**Example:**
```bash
curl http://localhost:8080/health
```

**Response:**
```json
{
  "status": "healthy"
}
```

---

## 🗄️ Database Schema

**Note:** All tables are automatically created by GORM on startup. No manual SQL needed!

### Users Table
```sql
CREATE TABLE users (
    id VARCHAR(255) PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    email_verified BOOLEAN DEFAULT false,
    name VARCHAR(255),
    given_name VARCHAR(255),
    family_name VARCHAR(255),
    picture TEXT,
    google_id VARCHAR(255) UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### OAuth Clients Table
```sql
CREATE TABLE oauth_clients (
    client_id VARCHAR(255) PRIMARY KEY,
    client_secret VARCHAR(255),
    client_name VARCHAR(255) NOT NULL,
    client_type VARCHAR(50) NOT NULL, -- 'public' or 'confidential'
    redirect_uris TEXT[] NOT NULL,    -- Uses pq.StringArray in Go
    grant_types TEXT[] NOT NULL,      -- Uses pq.StringArray in Go
    scope VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Auto-seeded development client:
-- client_id: 'demo-frontend'
-- client_secret: 'dev-secret'
-- client_type: 'public'
-- redirect_uris: ARRAY['http://localhost:3000/callback']
```

### Refresh Tokens Table
```sql
CREATE TABLE refresh_tokens (
    token VARCHAR(500) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id VARCHAR(255) NOT NULL,
    scope VARCHAR(500),
    expires_at TIMESTAMP NOT NULL,
    revoked BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Revoked Tokens Table
```sql
CREATE TABLE revoked_tokens (
    token_hash VARCHAR(64) PRIMARY KEY,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## 🏗️ Architecture

### Request Flow

```
Client Application
       ↓
   [API Input Layer]
   HTTP Handlers (authorize.go, token.go, userinfo.go, introspect.go)
       ↓
   [Business Logic Layer]
   OAuth Services (authcode, token_service, client_registry, pkce)
   User Auth (user/auth.go)
   Security (jwt, hasher, keys)
       ↓
   [Data Access Layer]
   Repositories (user_repo, client_repo, token_repo)
       ↓
   [Database]
   PostgreSQL/Neon
```

### Component Responsibilities

1. **HTTP Handlers** (`internal/http/handlers/`)
   - **Input**: Parse and validate HTTP requests
   - **Output**: Format and send HTTP responses
   - Handle API-level concerns (headers, status codes, etc.)

2. **OAuth Services** (`internal/oauth/`)
   - **authcode.go**: Manage authorization code lifecycle
   - **token_service.go**: Generate and refresh JWT tokens
   - **client_registry.go**: Validate OAuth clients
   - **pkce.go**: PKCE challenge/verifier validation

3. **Security** (`internal/security/`)
   - **jwt.go**: Sign and verify JWT tokens
   - **hasher.go**: Hash passwords and secrets
   - **keys.go**: Manage RSA key pairs

4. **Storage** (`internal/storage/`)
   - **user_repo.go**: User CRUD operations (sql.DB-based)
   - **client_repo.go**: OAuth client operations (GORM-based Storage struct)
   - **token_repo.go**: Token storage and revocation (sql.DB-based)
   - **db.go**: Database connection, GORM migrations, and auto-seeding

5. **User Auth** (`internal/user/`)
   - **auth.go**: User authentication and management

## 🔒 Security Features

- ✅ **PKCE Support** - Prevents authorization code interception attacks
- ✅ **JWT Signing** - HMAC-SHA256 token signing
- ✅ **Token Expiration** - Access tokens (1 hour), Refresh tokens (30 days)
- ✅ **Token Revocation** - Blacklist compromised tokens
- ✅ **CORS Protection** - Configurable CORS middleware
- ✅ **State Parameter** - CSRF protection in OAuth flow
- ✅ **Secure Random Generation** - Cryptographically secure tokens

## 🧪 Testing the Service

### 1. Use Auto-Seeded Client (Recommended)

The server automatically creates a `demo-frontend` client on startup - no manual setup needed!

### 2. Initiate OAuth Flow

Visit in browser (using auto-seeded client):
```
http://localhost:8080/authorize?client_id=demo-frontend&redirect_uri=http://localhost:3000/callback&response_type=code&state=random-state
```

### 3. Exchange Code for Tokens

After redirect, extract the `code` parameter and exchange it:

```bash
curl -X POST http://localhost:8080/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code" \
  -d "code=YOUR_CODE" \
  -d "redirect_uri=http://localhost:3000/callback" \
  -d "client_id=demo-frontend" \
  -d "client_secret=dev-secret"
```

### (Optional) Register Additional OAuth Clients

If you need additional test clients, use GORM or manual SQL:

```sql
INSERT INTO oauth_clients (
    client_id, 
    client_secret, 
    client_name, 
    client_type, 
    redirect_uris, 
    grant_types, 
    scope
) VALUES (
    'test-client',
    'test-secret',
    'Test Application',
    'confidential',
    ARRAY['http://localhost:3000/callback'],
    ARRAY['authorization_code', 'refresh_token'],
    'openid email profile'
);
```

### 4. Get User Info

```bash
curl http://localhost:8080/userinfo \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## 📚 Environment Variables Reference

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `GOOGLE_CLIENT_ID` | Google OAuth Client ID | ✅ Yes | - |
| `GOOGLE_CLIENT_SECRET` | Google OAuth Client Secret | ✅ Yes | - |
| `GOOGLE_REDIRECT_URL` | OAuth callback URL | No | `http://localhost:8080/callback` |
| `PORT` | Server port | No | `8080` |
| `JWT_SECRET` | Secret key for JWT signing | No | `default-jwt-secret...` |
| `DATABASE_URL` | PostgreSQL connection string | ✅ Yes | - |

## 🚦 Production Checklist

- [ ] Change `JWT_SECRET` to a strong random value
- [ ] Use PostgreSQL instead of in-memory storage for auth codes
- [ ] Implement Redis for auth code and session storage
- [ ] Enable HTTPS/TLS
- [ ] Set up proper CORS origins (not `*`)
- [ ] Implement rate limiting
- [ ] Add logging and monitoring
- [ ] Set up database backups
- [ ] Implement key rotation strategy
- [ ] Add comprehensive error handling
- [ ] Set up CI/CD pipeline
- [ ] Perform security audit