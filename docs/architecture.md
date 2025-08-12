# Tiris Backend System Architecture Document

## 1. Architecture Overview

### 1.1 System Context
Tiris Backend serves as the central data management microservice in the Tiris quantitative trading ecosystem. It provides RESTful APIs for user management, exchange integration, and trading data operations while maintaining data consistency and security.

### 1.2 Architecture Style
- **Microservice Architecture**: Single-responsibility service focused on data management
- **Layered Architecture**: Clean separation between API, business logic, and data layers
- **RESTful API Design**: Standard HTTP methods and status codes for external communication

## 2. High-Level Architecture

```
┌─────────────────────────────────────────────────────────┐
│                  Tiris Ecosystem                       │
├─────────────────────────────────────────────────────────┤
│      tiris-portal      │      tiris-bot                │
│       (Frontend)       │      (Trading)                │
└────────┬───────────────┴──────────┬────────────────────┘
         │ HTTP                     │ NATS Events
         │                          │
         ▼                          ▼
┌─────────────────────────────────────────────────────────┐
│                 Tiris Backend                           │
│               (Data Management)                         │
│  ┌─────────────────┐  ┌─────────────────────────────┐   │
│  │   HTTP APIs     │  │   NATS Event Consumer       │   │
│  │  (User Ops)     │  │   (Trading Events)          │   │
│  └─────────────────┘  └─────────────────────────────┘   │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────┐
│            PostgreSQL + TimescaleDB                    │
│              (Data Storage)                             │
└─────────────────────────────────────────────────────────┘
```

## 3. System Components

### 3.1 API Layer
**Responsibilities:**
- HTTP request routing and handling
- Request/response validation
- Authentication and authorization
- Rate limiting and throttling
- API documentation (OpenAPI/Swagger)

**Components:**
- HTTP Router (Gorilla Mux or Gin)
- Middleware stack (Auth, Logging, CORS)
- Request validators
- Response serializers

### 3.2 Business Logic Layer
**Responsibilities:**
- Core business rules implementation
- Data transformation and validation
- Integration with external services
- Transaction management

**Components:**
- User Service
- Exchange Service
- Sub-account Service
- Transaction Service
- Trading Log Service

### 3.3 Data Access Layer
**Responsibilities:**
- Database connection management
- SQL query execution
- Data mapping and serialization
- Connection pooling

**Components:**
- Repository pattern implementations
- Database connection pool
- Query builders
- Transaction managers

### 3.4 Message Queue Layer
**Responsibilities:**
- NATS JetStream event consumption
- Trading event processing and validation
- Event ordering and deduplication
- Dead letter queue handling

**Components:**
- NATS consumer clients
- Event processors
- Message validators
- Error handlers

### 3.5 External Integration Layer
**Responsibilities:**
- OAuth provider integration
- Exchange API communication
- Third-party service calls

**Components:**
- OAuth clients (Google, WeChat)
- Exchange API clients
- HTTP clients with retry logic

## 4. Detailed Component Design

### 4.1 API Layer Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    API Gateway                          │
├─────────────────────────────────────────────────────────┤
│  Middleware Stack:                                      │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐      │
│  │   CORS      │ │    Auth     │ │   Logging   │      │
│  └─────────────┘ └─────────────┘ └─────────────┘      │
├─────────────────────────────────────────────────────────┤
│  Route Handlers:                                        │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐      │
│  │    User     │ │  Exchange   │ │Sub-account  │      │
│  │  Handler    │ │   Handler   │ │   Handler   │      │
│  └─────────────┘ └─────────────┘ └─────────────┘      │
│  ┌─────────────┐ ┌─────────────┐                      │
│  │Transaction  │ │Trading Log  │                      │
│  │  Handler    │ │   Handler   │                      │
│  └─────────────┘ └─────────────┘                      │
└─────────────────────────────────────────────────────────┘
```

### 4.2 Business Logic Layer Architecture

```
┌─────────────────────────────────────────────────────────┐
│                 Business Services                       │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐      │
│  │    User     │ │  Exchange   │ │Sub-account  │      │
│  │  Service    │ │   Service   │ │   Service   │      │
│  └─────────────┘ └─────────────┘ └─────────────┘      │
│  ┌─────────────┐ ┌─────────────┐                      │
│  │Transaction  │ │Trading Log  │                      │
│  │  Service    │ │   Service   │                      │
│  └─────────────┘ └─────────────┘                      │
├─────────────────────────────────────────────────────────┤
│                 Shared Components                      │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐      │
│  │  Validator  │ │ Encryption  │ │   Logger    │      │
│  └─────────────┘ └─────────────┘ └─────────────┘      │
└─────────────────────────────────────────────────────────┘
```

### 4.3 Message Queue Layer Architecture

```
┌─────────────────────────────────────────────────────────┐
│                NATS Event Processing                    │
├─────────────────────────────────────────────────────────┤
│  Event Streams:                                         │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐      │
│  │trading.     │ │trading.     │ │trading.     │      │
│  │orders.*     │ │balance.*    │ │errors       │      │
│  └─────────────┘ └─────────────┘ └─────────────┘      │
├─────────────────────────────────────────────────────────┤
│  Event Processors:                                      │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐      │
│  │Order Event  │ │Balance Event│ │Error Event  │      │
│  │ Processor   │ │ Processor   │ │ Processor   │      │
│  └─────────────┘ └─────────────┘ └─────────────┘      │
├─────────────────────────────────────────────────────────┤
│  Message Handling:                                      │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐      │
│  │Deduplication│ │ Validation  │ │Dead Letter  │      │
│  │   Handler   │ │   Handler   │ │   Queue     │      │
│  └─────────────┘ └─────────────┘ └─────────────┘      │
└─────────────────────────────────────────────────────────┘
```

### 4.4 Data Access Layer Architecture

```
┌─────────────────────────────────────────────────────────┐
│                Repository Pattern                       │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐      │
│  │    User     │ │  Exchange   │ │Sub-account  │      │
│  │ Repository  │ │ Repository  │ │ Repository  │      │
│  └─────────────┘ └─────────────┘ └─────────────┘      │
│  ┌─────────────┐ ┌─────────────┐                      │
│  │Transaction  │ │Trading Log  │                      │
│  │ Repository  │ │ Repository  │                      │
│  └─────────────┘ └─────────────┘                      │
├─────────────────────────────────────────────────────────┤
│              Database Connection                        │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐      │
│  │  Connection │ │Query Builder│ │ Transaction │      │
│  │    Pool     │ │             │ │   Manager   │      │
│  └─────────────┘ └─────────────┘ └─────────────┘      │
└─────────────────────────────────────────────────────────┘
```

## 5. Data Architecture

### 5.1 Database Schema Overview

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│    users    │    │  exchanges  │    │sub_accounts │
│             │◄──┐│             │◄──┐│             │
│ - id        │   ││ - id        │   ││ - id        │
│ - username  │   ││ - user_id   │   ││ - user_id   │
│ - email     │   ││ - name      │   ││ - exchange_id│
│ - settings  │   ││ - type      │   ││ - name      │
│ - info      │   ││ - api_key   │   ││ - symbol    │
└─────────────┘   ││ - info      │   ││ - balance   │
                  │└─────────────┘   ││ - info      │
                  │                  │└─────────────┘
┌─────────────┐   │┌─────────────┐   │
│trading_logs │   ││transactions │   │
│             │   ││             │   │
│ - id        │   ││ - id        │   │
│ - user_id   │───┘│ - user_id   │───┘
│ - exchange_id    │ - exchange_id
│ - timestamp │    │ - sub_account_id
│ - type      │    │ - timestamp │
│ - source    │    │ - direction │
│ - message   │    │ - amount    │
│ - info      │    │ - info      │
└─────────────┘    └─────────────┘
```

### 5.2 JSON Extensibility Design

**Universal Info Column:**
- All database tables include an `info` JSONB column for flexible data storage
- Enables schema evolution without database migrations
- Supports varying data requirements across different exchanges and use cases
- GIN indexes on info columns for efficient JSON queries

**Benefits:**
- Rapid feature development without schema changes
- Exchange-specific data storage (API limits, permissions, etc.)
- User preference storage beyond standard settings
- Trading strategy parameters and metadata storage
- Audit trail and debugging information

### 5.3 Time-Series Data Optimization

**TimescaleDB Features:**
- Automatic partitioning for transactions and trading_logs tables
- Continuous aggregates for performance metrics
- Compression for historical data
- Retention policies for old data

**Indexing Strategy:**
- Primary keys on all tables
- Composite indexes on (user_id, timestamp) for time-series queries
- Indexes on foreign key relationships
- Partial indexes for frequently filtered data

## 6. Event-Driven Architecture

### 6.1 NATS Event Flow

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   tiris-bot │    │    NATS     │    │tiris-backend│
│             │    │ JetStream   │    │             │
└─────┬───────┘    └─────┬───────┘    └─────┬───────┘
      │                  │                  │
      │ 1. Trading Event │                  │
      ├─────────────────►│                  │
      │                  │                  │
      │ 2. Event Stored  │                  │
      │◄─────────────────┤                  │
      │                  │                  │
      │                  │ 3. Event Deliver │
      │                  ├─────────────────►│
      │                  │                  │
      │                  │ 4. Process Event │
      │                  │◄─────────────────┤
      │                  │                  │
      │                  │ 5. Ack Message   │
      │                  │◄─────────────────┤
```

### 6.2 Event Types and Processing

**Order Events:**
- `trading.orders.created` - Order placed on exchange
- `trading.orders.filled` - Order executed (full/partial)
- `trading.orders.cancelled` - Order cancelled
- `trading.orders.failed` - Order execution failed

**Balance Events:**
- `trading.balance.updated` - Sub-account balance changed
- `trading.balance.locked` - Funds locked for order
- `trading.balance.unlocked` - Funds released from order

**System Events:**
- `trading.errors` - Trading system errors
- `trading.signals` - Trading strategy signals
- `trading.heartbeat` - Bot health status

### 6.3 Event Processing Guarantees

**Ordering:**
- Events processed in order per user/sub-account
- Uses NATS JetStream consumer groups
- Partitioned by user_id for parallel processing

**Reliability:**
- At-least-once delivery guarantee
- Idempotent event processing with deduplication
- Dead letter queue for failed events
- Automatic retry with exponential backoff

**Durability:**
- Events persisted to disk by NATS JetStream
- Configurable retention policies
- Event replay capability for recovery

## 7. Security Architecture

### 7.1 Authentication Flow

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Client    │    │    OAuth    │    │   Backend   │
│             │    │  Provider   │    │             │
└─────┬───────┘    └─────┬───────┘    └─────┬───────┘
      │                  │                  │
      │ 1. Auth Request  │                  │
      ├─────────────────►│                  │
      │                  │                  │
      │ 2. Auth Code     │                  │
      │◄─────────────────┤                  │
      │                  │                  │
      │ 3. Token Exchange│                  │
      ├─────────────────────────────────────►│
      │                  │                  │
      │ 4. User Info     │                  │
      │◄─────────────────┼──────────────────┤
      │                  │                  │
      │ 5. JWT Token     │                  │
      │◄─────────────────────────────────────┤
```

### 6.2 Data Protection

**Encryption:**
- API keys and secrets encrypted using AES-256
- TLS 1.3 for all external communications
- Environment-based key management

**Access Control:**
- JWT-based authentication for API access
- Role-based permissions (User, Admin)
- Resource-level authorization checks

## 7. Performance Architecture

### 7.1 Caching Strategy

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│    Client   │    │   Backend   │    │  Database   │
│             │    │             │    │             │
└─────┬───────┘    └─────┬───────┘    └─────┬───────┘
      │                  │                  │
      │ 1. Request       │                  │
      ├─────────────────►│                  │
      │                  │ 2. Check Cache   │
      │                  ├─────────────────►│
      │                  │                  │
      │                  │ 3. Cache Miss    │
      │                  │◄─────────────────┤
      │                  │                  │
      │                  │ 4. Query DB      │
      │                  ├─────────────────►│
      │                  │                  │
      │                  │ 5. Results       │
      │                  │◄─────────────────┤
      │                  │                  │
      │ 6. Response      │ 7. Update Cache  │
      │◄─────────────────┤                  │
```

**Cache Layers:**
- Application-level caching for user sessions
- Database query result caching
- Static data caching (exchange configurations)

### 7.2 Connection Pooling

**Database Connections:**
- Connection pool size: 10-50 connections
- Connection timeout: 30 seconds
- Idle connection cleanup: 5 minutes
- Health checks every 30 seconds

## 8. Deployment Architecture

### 8.1 Docker Container Structure

```
┌─────────────────────────────────────────────────────────┐
│                Docker Container                         │
├─────────────────────────────────────────────────────────┤
│  Base Image: golang:1.21-alpine                        │
│                                                         │
│  Application Layer:                                     │
│  ┌─────────────────────────────────────────────────────┐│
│  │  tiris-backend binary                               ││
│  │  Configuration files                                ││
│  │  SSL certificates                                   ││
│  └─────────────────────────────────────────────────────┘│
│                                                         │
│  Exposed Ports: 8080 (HTTP)                           │
│  Environment Variables: DB_HOST, OAUTH_KEYS, etc.     │
└─────────────────────────────────────────────────────────┘
```

### 8.2 Kubernetes Deployment (Future)

```
┌─────────────────────────────────────────────────────────┐
│                    Namespace: tiris                     │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐      │
│  │  Ingress    │ │   Service   │ │ Deployment  │      │
│  │ Controller  │ │  (ClusterIP)│ │  (3 Pods)   │      │
│  └─────────────┘ └─────────────┘ └─────────────┘      │
│                                                         │
│  ┌─────────────┐ ┌─────────────┐                      │
│  │ ConfigMap   │ │   Secret    │                      │
│  │ (App Config)│ │(DB Creds)   │                      │
│  └─────────────┘ └─────────────┘                      │
└─────────────────────────────────────────────────────────┘
```

## 9. Monitoring and Observability

### 9.1 Logging Architecture

**Log Levels:**
- ERROR: System errors, API failures
- WARN: Validation errors, rate limits
- INFO: API requests, business operations
- DEBUG: Detailed execution flow

**Log Format:**
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "service": "tiris-backend",
  "trace_id": "abc123",
  "user_id": "user123",
  "message": "User login successful",
  "metadata": {...}
}
```

### 9.2 Metrics Collection

**Key Metrics:**
- API response times (P50, P95, P99)
- Request count by endpoint
- Error rates by type
- Database connection pool usage
- Memory and CPU utilization

**Health Checks:**
- Liveness probe: /health/live
- Readiness probe: /health/ready
- Database connectivity check
- External service dependency checks

## 10. Error Handling Architecture

### 10.1 Error Categories

**Business Errors:**
- Validation errors (400)
- Authentication errors (401)
- Authorization errors (403)
- Resource not found (404)

**System Errors:**
- Database connection errors (503)
- External service timeouts (504)
- Internal server errors (500)

### 10.2 Error Response Format

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "User-friendly error message",
    "details": "Technical details for debugging",
    "trace_id": "abc123",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

## 11. Scalability Considerations

### 11.1 Horizontal Scaling
- Stateless service design
- Database connection pooling
- Load balancer compatibility
- Session management externalization

### 11.2 Vertical Scaling
- Memory-efficient data structures
- Optimized database queries
- Connection reuse
- Garbage collection tuning

### 11.3 Database Scaling
- Read replicas for query performance
- Partitioning strategies for large tables
- Index optimization
- Query caching

## 12. Future Architecture Evolution

### 12.1 Microservice Decomposition
- Separate user management service
- Dedicated exchange integration service
- Independent analytics service

### 12.2 Event-Driven Architecture
- Message queue integration (NATS, Kafka)
- Event sourcing for trading operations
- Real-time data streaming

### 12.3 Advanced Features
- GraphQL API support
- WebSocket connections for real-time data
- Machine learning integration APIs
- Multi-tenant architecture support

## 13. Resources and Domain Architecture

### 13.1 Domain Configuration
- **Production Domain**: `tiris.ai`
- **Development Domain**: `dev.tiris.ai`
- **API Subdomains**: 
  - Production: `api.tiris.ai`
  - Development: `api.dev.tiris.ai`

### 13.2 Service Endpoint Mapping

```
Production (tiris.ai):
├── Frontend: https://tiris.ai
├── API: https://api.tiris.ai/v1
├── Admin: https://admin.tiris.ai
└── Docs: https://docs.tiris.ai

Development (dev.tiris.ai):
├── Frontend: https://dev.tiris.ai
├── API: https://api.dev.tiris.ai/v1
├── Admin: https://admin.dev.tiris.ai
└── Docs: https://docs.dev.tiris.ai
```

### 13.3 SSL/TLS Architecture
- **Certificate Management**: Let's Encrypt with auto-renewal
- **TLS Version**: 1.2 minimum, 1.3 preferred
- **HSTS**: Enabled with 1-year max-age
- **Certificate Transparency**: Enabled for all domains

### 13.4 CDN and Load Balancing
- **Frontend Assets**: CDN distribution for static files
- **API Load Balancing**: Geographic distribution
- **Failover Strategy**: Multi-region deployment capability
- **Health Checks**: Endpoint monitoring across all services

### 13.5 Environment Isolation
- **Network Segregation**: Separate VPCs for prod/dev
- **Database Isolation**: Dedicated instances per environment
- **Secret Management**: Environment-specific secret stores
- **Monitoring**: Separate dashboards and alerting per environment