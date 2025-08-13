# GitHub Actions CI/CD Pipeline

This directory contains the complete CI/CD pipeline configuration for the Tiris Backend project using GitHub Actions.

## Workflows Overview

### 1. CI/CD Pipeline (`ci.yml`)

**Triggers:**
- Push to `master`, `main`, or `develop` branches
- Pull requests to `master` or `main` branches

**Jobs:**

#### Lint Job
- Code formatting with `gofmt`
- Linting with `golangci-lint`
- Go module consistency check

#### Test Job
- **Services:** PostgreSQL + TimescaleDB, Redis, NATS JetStream
- **Test Coverage:** Minimum 70% threshold with Codecov integration
- **Database:** Automated migrations and TimescaleDB extensions
- **Environment:** Full test environment with all dependencies

#### Build Job
- Cross-platform binary compilation
- Build artifact archiving
- Binary validation

#### Docker Job
- Multi-platform Docker image builds (linux/amd64, linux/arm64)
- Automated tagging and pushing to registries
- Build cache optimization

#### Security Job
- **Gosec:** Static security analysis
- **Trivy:** Vulnerability scanning
- **SARIF:** Security findings upload to GitHub

#### Integration Job
- End-to-end testing with Docker Compose
- Real service integration testing
- Health check validation

### 2. Release Pipeline (`release.yml`)

**Triggers:**
- Git tags matching `v*.*.*` pattern
- Manual workflow dispatch

**Features:**

#### Release Management
- Automatic changelog generation
- GitHub release creation
- Pre-release detection (alpha, beta, rc)

#### Binary Distribution
- **Platforms:** Linux, macOS, Windows
- **Architectures:** amd64, arm64
- **Artifacts:** Server and migrate binaries with checksums
- **Versioning:** Embedded version, build time, and git commit

#### Container Images
- Production-optimized multi-stage Dockerfile
- Multi-platform container builds
- Registry publishing (GitHub Container Registry + Docker Hub)
- Semantic versioning tags

#### Helm Charts
- Kubernetes deployment charts
- Configurable values for different environments
- Chart packaging and distribution

### 3. Dependency Updates (`dependency-update.yml`)

**Schedule:** Every Monday at 9:00 AM UTC

**Features:**

#### Go Dependencies
- Automated minor/patch version updates
- Test validation with updated dependencies
- Pull request creation for review

#### GitHub Actions
- Action version updates to latest releases
- Compatibility validation
- Security and performance improvements

#### Security Auditing
- Vulnerability scanning with `govulncheck`
- Automated security issue creation
- High-priority security alerts

## Configuration

### Required Secrets

```yaml
# Container Registry
DOCKER_USERNAME          # Docker Hub username (optional)
DOCKER_PASSWORD          # Docker Hub password (optional)
CODECOV_TOKEN            # Codecov integration token (optional)

# GitHub token is automatically provided
GITHUB_TOKEN             # Automatically available
```

### Environment Variables

```yaml
GO_VERSION: '1.23'           # Go version for all jobs
POSTGRES_VERSION: '15'       # PostgreSQL version
REDIS_VERSION: '7-alpine'    # Redis version
NATS_VERSION: 'alpine'       # NATS version
```

## Workflow Features

### 🔒 Security First
- Static code analysis with Gosec
- Dependency vulnerability scanning
- SARIF security report integration
- Automated security issue creation

### 🧪 Comprehensive Testing
- Unit tests with race condition detection
- Integration tests with real services
- Test coverage reporting and enforcement
- Multi-environment testing support

### 📦 Artifact Management
- Cross-platform binary builds
- Container image multi-architecture support
- Helm chart packaging
- Checksum generation for integrity

### 🚀 Deployment Ready
- Docker Compose development environment
- Kubernetes-ready container images
- Helm chart for production deployment
- Health check validation

### 🔄 Automation
- Dependency update automation
- Security vulnerability monitoring
- Release management automation
- Pull request creation for reviews

## Usage Examples

### Triggering a Release

```bash
# Create and push a release tag
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0

# Or trigger manually via GitHub UI
# Actions → Release → Run workflow → Enter tag (e.g., v1.0.0)
```

### Local Testing

```bash
# Run tests locally with same environment
docker-compose -f docker-compose.dev.yml up -d
docker-compose -f docker-compose.dev.yml run --rm migrate
docker-compose -f docker-compose.dev.yml --profile setup run --rm nats-setup

# Run tests
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Development Workflow

```bash
# Start development environment
docker-compose -f docker-compose.dev.yml up -d

# Make changes to code
# Tests run automatically on push/PR

# View pipeline status
gh workflow list
gh workflow view ci.yml
gh run list
```

## Monitoring and Alerts

### Status Badges

Add to README.md:

```markdown
[![CI/CD Pipeline](https://github.com/your-org/tiris-backend/actions/workflows/ci.yml/badge.svg)](https://github.com/your-org/tiris-backend/actions/workflows/ci.yml)
[![Security Scan](https://github.com/your-org/tiris-backend/actions/workflows/ci.yml/badge.svg)](https://github.com/your-org/tiris-backend/security)
[![Coverage](https://codecov.io/gh/your-org/tiris-backend/branch/master/graph/badge.svg)](https://codecov.io/gh/your-org/tiris-backend)
```

### Notifications

The pipeline includes automated notifications for:
- ✅ Successful builds and deployments
- ❌ Failed tests or security issues  
- 🔄 Dependency updates available
- 🚨 Security vulnerabilities detected

## Troubleshooting

### Common Issues

1. **Test Failures in CI**
   - Check service health in workflow logs
   - Verify database migrations are applied
   - Ensure test environment variables are set

2. **Docker Build Failures**
   - Check Dockerfile syntax and dependencies
   - Verify multi-platform build compatibility
   - Review build cache issues

3. **Security Scan Issues**
   - Update vulnerable dependencies
   - Review Gosec findings for false positives
   - Check Trivy vulnerability database

4. **Release Failures**
   - Verify semantic versioning tag format
   - Check binary build compatibility
   - Ensure registry credentials are valid

### Debug Steps

```bash
# Check workflow runs
gh run list --workflow=ci.yml

# View specific run details
gh run view <run-id>

# Download artifacts for inspection
gh run download <run-id>

# Re-run failed jobs
gh run rerun <run-id>
```

## Best Practices

### Branch Protection

Configure branch protection rules:
- Require PR reviews
- Require status checks (CI pipeline)
- Require up-to-date branches
- Restrict pushes to main/master

### Security

- Regularly review dependency updates
- Monitor security advisories
- Keep GitHub Actions up to date
- Use least-privilege principles

### Performance

- Use action caching effectively
- Minimize job dependencies
- Run jobs in parallel when possible
- Monitor workflow execution times

This CI/CD pipeline provides enterprise-grade automation, security, and reliability for the Tiris Backend project while maintaining developer productivity and code quality.