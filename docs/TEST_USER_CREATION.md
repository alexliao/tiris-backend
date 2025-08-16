# Test User Creation Guide

This guide explains how to create test users for the Tiris Backend application, which uses OAuth-only authentication (Google and WeChat).

## Quick Start

### Create a Simple Test User
```bash
# Using the script directly
./scripts/create-test-user.sh --name "John Doe"

# Using Makefile
make create-test-user ARGS="--name 'John Doe'"
```

### Create a Custom Test User
```bash
# With custom details
./scripts/create-test-user.sh \
  --name "Jane Smith" \
  --username "jane_dev" \
  --email "jane@example.com" \
  --provider "wechat"

# With custom expiration
./scripts/create-test-user.sh \
  --name "Test User" \
  --expiry "6 months"
```

## Script Options

| Option | Short | Description | Default | Required |
|--------|-------|-------------|---------|----------|
| `--name` | `-n` | User's display name | - | ✅ |
| `--username` | `-u` | Username (auto-generated if not provided) | Generated from name | ❌ |
| `--email` | `-e` | Email address (auto-generated if not provided) | `{username}@tiris.local` | ❌ |
| `--provider` | `-p` | OAuth provider (`google` or `wechat`) | `google` | ❌ |
| `--expiry` | `-t` | Token expiration period | `1 year` | ❌ |
| `--help` | `-h` | Show help message | - | ❌ |

## Examples

### Basic Examples
```bash
# Minimal - just provide a name
./scripts/create-test-user.sh --name "Alice Developer"

# With custom username
./scripts/create-test-user.sh --name "Bob Smith" --username "bob_dev"

# With custom email
./scripts/create-test-user.sh --name "Carol Jones" --email "carol@mycompany.com"
```

### Provider Examples
```bash
# Google OAuth user (default)
./scripts/create-test-user.sh --name "Google User"

# WeChat OAuth user
./scripts/create-test-user.sh --name "微信用户" --provider "wechat"
```

### Expiration Examples
```bash
# Long-term debugging user (1 year - default)
./scripts/create-test-user.sh --name "Debug User Long"

# Short-term testing (6 months)
./scripts/create-test-user.sh --name "Debug User Short" --expiry "6 months"

# Very short-term (1 day)
./scripts/create-test-user.sh --name "Temp User" --expiry "1 day"
```

### Makefile Examples
```bash
# Basic usage through make
make create-test-user ARGS="--name 'Development User'"

# Complex usage through make
make create-test-user ARGS="--name 'API Tester' --provider wechat --expiry '3 months'"
```

## Understanding the Output

After creating a user, the script provides:

```
===============================================
📋 USER DETAILS
===============================================
Name:          Alice Developer
Username:      alice_developer
Email:         alice_developer@tiris.local
User ID:       ef72227a-0cf7-4c0a-8073-29a7ccdc3bca
Provider:      google
Access Token:  google_token_alice_developer_1755306482
Expires:       1 year from now

===============================================
🔧 API TESTING
===============================================
Use this Authorization header in your API requests:

Authorization: Bearer google_token_alice_developer_1755306482

Example curl command:
curl -H "Authorization: Bearer google_token_alice_developer_1755306482" \
     http://localhost:8080/v1/users/profile

===============================================
```

## Using Test Users for API Testing

### With curl
```bash
curl -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
     -X GET \
     http://localhost:8080/v1/users/me
```

### With Postman
1. Set Authorization Type to "Bearer Token"
2. Paste the access token from the script output
3. Make requests to `http://localhost:8080/v1/*`

### With JavaScript/Fetch
```javascript
fetch('http://localhost:8080/v1/users/me', {
  headers: {
    'Authorization': 'Bearer YOUR_ACCESS_TOKEN',
    'Content-Type': 'application/json'
  }
})
.then(response => response.json())
.then(data => console.log(data));
```

## Database Structure

The script creates records in two tables:

### users table
- `id`: UUID primary key
- `username`: Unique username
- `email`: User email
- `avatar`: Profile image URL
- `settings`: JSON user preferences
- `info`: JSON user metadata (includes `test_user: true`)

### oauth_tokens table
- `id`: UUID primary key
- `user_id`: Foreign key to users table
- `provider`: OAuth provider (`google` or `wechat`)
- `provider_user_id`: Unique ID from OAuth provider
- `access_token`: Bearer token for API authentication
- `refresh_token`: Token for refreshing access
- `expires_at`: Token expiration timestamp
- `info`: JSON OAuth profile data (includes `test_user: true`)

## Prerequisites

1. **PostgreSQL Running**: The database must be accessible
   ```bash
   docker compose -f docker-compose.dev.yml up -d postgres
   ```

2. **Database Migrations Applied**: Tables must exist
   ```bash
   make migrate-up
   ```

3. **Script Permissions**: The script must be executable
   ```bash
   chmod +x scripts/create-test-user.sh
   ```

## Troubleshooting

### "Cannot connect to PostgreSQL"
```bash
# Check if PostgreSQL is running
docker compose -f docker-compose.dev.yml ps postgres

# Start PostgreSQL if needed
docker compose -f docker-compose.dev.yml up -d postgres

# Wait a moment for startup, then retry
```

### "duplicate key value violates unique constraint"
The username already exists. Either:
- Use a different name: `--name "Different Name"`
- Specify a unique username: `--username "unique_username"`
- Check existing users: `docker exec tiris-postgres-dev psql -U tiris_user -d tiris_dev -c "SELECT username FROM users;"`

### "Required environment variable not set"
This error comes from the main application, not the test user script. The script works independently of OAuth configuration.

## Advanced Usage

### Batch User Creation
```bash
#!/bin/bash
names=("Alice Dev" "Bob Test" "Carol QA" "David Debug")
for name in "${names[@]}"; do
    ./scripts/create-test-user.sh --name "$name"
done
```

### Different Environments
```bash
# Development users
./scripts/create-test-user.sh --name "Dev User" --email "dev@tiris.local"

# Staging users  
./scripts/create-test-user.sh --name "Staging User" --email "staging@tiris.local"

# Load testing users
./scripts/create-test-user.sh --name "Load Test User" --expiry "1 day"
```

## Security Considerations

1. **Test Users Only**: These users are marked with `test_user: true` in their metadata
2. **Local Emails**: Default emails use `@tiris.local` domain to avoid conflicts
3. **Fake Tokens**: Access tokens are clearly identifiable as test tokens
4. **Development Only**: This script should only be used in development/testing environments

## Integration with Development Workflow

Add to your development setup:

```bash
# In your development setup script
make migrate-up
./scripts/create-test-user.sh --name "Dev User" --username "dev"
make run
```

This ensures you always have a test user available for API development and testing.