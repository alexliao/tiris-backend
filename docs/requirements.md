# Tiris Backend Requirements Document

## 1. Project Overview

### 1.1 Product Vision
Tiris Backend is a RESTful microservice that serves as the data management layer for the Tiris quantitative trading ecosystem. It manages user accounts, trading data, and provides APIs for the frontend portal while ensuring secure, scalable, and efficient data operations.

### 1.2 Scope
This document defines the functional and non-functional requirements for the Tiris Backend service, including user management, trading platform integration, sub-account management, transaction tracking, and trading log management.

## 2. Functional Requirements

### 2.1 User Management (FR-UM)

**FR-UM-001: OAuth Authentication**
- The system MUST support OAuth authentication with Google
- The system MUST support OAuth authentication with WeChat
- The system MUST automatically create user records for new OAuth users
- The system SHOULD support additional OAuth providers in the future

**FR-UM-002: User Profile Management**
- Users MUST be able to view their profile information
- Users MUST be able to update their profile information (username, email, avatar)
- Users MUST be able to update their settings (stored as JSON)
- Users MUST be able to disable their accounts
- Users MUST be able to remove their accounts

**FR-UM-003: User Data Queries**
- The system MUST provide APIs to query user information
- The system MUST enforce proper authorization for user data access

### 2.2 Trading Platform Management (FR-TPM)

**FR-TPM-001: Trading Platform Binding**
- Users MUST be able to bind multiple trading platforms to their account
- Users MUST provide trading platform type, name, API key, and API secret
- The system MUST validate trading platform credentials before binding
- The system MUST create a virtual trading platform for each user by default for simulation trading

**FR-TPM-002: Trading Platform Configuration**
- Users MUST be able to modify trading platform configurations
- Users MUST be able to unbind (remove) trading platforms
- The system MUST secure API keys and secrets using encryption

**FR-TPM-003: Trading Platform Data Queries**
- Users MUST be able to query their trading platform information
- The system MUST not expose sensitive credentials in API responses

### 2.3 Sub-account Management (FR-SM)

**FR-SM-001: Sub-account Creation**
- Users MUST be able to create multiple sub-accounts within an trading platform
- Users MUST specify initial balance, symbol, and name for each sub-account
- The system MUST validate that sub-account balances do not exceed available trading platform funds

**FR-SM-002: Sub-account Operations**
- Users MUST be able to deposit funds into sub-accounts
- Users MUST be able to withdraw funds from sub-accounts
- Users MUST be able to modify sub-account information (name)
- Users MUST be able to delete sub-accounts

**FR-SM-003: Sub-account Queries**
- Users MUST be able to query sub-account information
- The system MUST provide real-time balance information

### 2.4 Transaction Management (FR-TM)

**FR-TM-001: Transaction Recording**
- The system MUST automatically create transactions based on trading logs
- Transactions MUST record timestamp, direction (debit/credit), reason, amount, closing balance, price, and quote symbol
- The system MUST maintain transaction integrity and consistency

**FR-TM-002: Transaction Queries**
- Users MUST be able to query transaction history
- The system MUST support filtering by date range, sub-account, and transaction type
- The system MUST provide pagination for large result sets

### 2.5 Trading Log Management (FR-TL)

**FR-TL-001: Trading Log Creation**
- The system MUST accept trading logs from external sources (bots via NATS, manual trading via API)
- Trading logs MUST include timestamp, type, source, message, and reference IDs
- The system MUST validate trading log data before storage

**FR-TL-002: Trading Log Operations**
- Users MUST be able to query trading logs
- Administrators MUST be able to delete trading logs
- The system MUST support real-time log streaming for monitoring

### 2.6 Message Queue Processing (FR-MQ)

**FR-MQ-001: Event Consumption**
- The system MUST consume trading events from NATS JetStream
- The system MUST process events in order per user/sub-account
- The system MUST handle event replay and duplicate detection

**FR-MQ-002: Event Processing**
- The system MUST automatically create trading logs from bot events
- The system MUST automatically create transactions from order execution events
- The system MUST update sub-account balances based on trading events
- The system MUST validate all event data before processing

**FR-MQ-003: Reliability and Error Handling**
- The system MUST acknowledge messages only after successful processing
- The system MUST implement dead letter queue for failed events
- The system MUST provide event processing metrics and monitoring
- The system MUST support event replay for recovery scenarios

## 3. Non-Functional Requirements

### 3.1 Performance (NFR-P)

**NFR-P-001: Response Time**
- API responses MUST complete within 500ms for 95% of requests
- Database queries MUST be optimized for time-series data operations
- The system MUST handle concurrent requests efficiently

**NFR-P-002: Throughput**
- The system MUST support at least 1000 concurrent users
- The system MUST handle at least 10,000 transactions per minute
- Trading log ingestion MUST support high-frequency data streams
- NATS message processing MUST handle at least 50,000 events per minute
- Event processing latency MUST be under 100ms for 95% of events

### 3.2 Scalability (NFR-S)

**NFR-S-001: Horizontal Scaling**
- The system MUST be designed for horizontal scaling using Kubernetes
- Database operations MUST be optimized for distributed environments
- The system MUST support load balancing across multiple instances

### 3.3 Security (NFR-SEC)

**NFR-SEC-001: Data Protection**
- All API keys and secrets MUST be encrypted at rest
- The system MUST use HTTPS for all external communications
- User data MUST be protected according to data privacy regulations

**NFR-SEC-002: Authentication & Authorization**
- All API endpoints MUST require proper authentication
- The system MUST implement role-based access control
- OAuth tokens MUST be properly validated and refreshed

### 3.4 Reliability (NFR-R)

**NFR-R-001: Availability**
- The system MUST maintain 99.9% uptime
- The system MUST implement proper error handling and recovery
- Database operations MUST be transactional where appropriate

**NFR-R-002: Data Integrity**
- All financial data MUST maintain ACID properties
- The system MUST implement backup and disaster recovery procedures
- Data validation MUST prevent corruption and inconsistencies

### 3.5 Maintainability (NFR-M)

**NFR-M-001: Code Quality**
- The system MUST follow Go best practices and conventions
- Code coverage MUST be at least 80%
- The system MUST use test-driven development approaches

**NFR-M-002: Monitoring & Logging**
- The system MUST provide comprehensive logging for debugging
- The system MUST expose metrics for monitoring and alerting
- All API calls MUST be logged with appropriate detail levels

## 4. Technical Constraints

### 4.1 Technology Stack
- Programming Language: Go (Golang)
- Database: PostgreSQL with TimescaleDB extension
- Message Queue: NATS JetStream
- API Style: RESTful (user-facing) + Event-driven (internal)
- Deployment: Docker containers
- Future Orchestration: Kubernetes

### 4.2 Integration Requirements
- OAuth providers: Google, WeChat
- Cryptocurrency trading platforms: Binance, Kraken, Gate.io
- Internal services: tiris-portal (HTTP), tiris-bot (NATS)
- Message queue: NATS JetStream for event processing

### 4.3 Data Requirements
- Time-series data optimization for trading logs and transactions
- JSON storage support for flexible data structures
- All database tables MUST include an `info` column (JSON type) for extended and variable information
- Primary key `id` column required for all tables
- Proper indexing for performance-critical queries

## 5. Acceptance Criteria

### 5.1 User Management
- New users can sign in using Google OAuth and have accounts created automatically
- Users can update their profile information successfully
- Users can disable/enable their accounts

### 5.2 Trading Platform Management
- Users can bind trading platforms with valid API credentials
- Invalid credentials are properly rejected with clear error messages
- Trading platform data is securely stored and properly encrypted

### 5.3 Sub-account Management
- Users can create sub-accounts with specified balances
- Sub-account operations (deposit/withdraw) update balances correctly
- Balance constraints are enforced to prevent overdrafts

### 5.4 Transaction & Trading Log Management
- Trading logs automatically generate corresponding transactions
- All financial calculations are accurate to the precision required
- Query performance meets specified response time requirements

## 6. Assumptions and Dependencies

### 6.1 Assumptions
- Trading platform APIs will remain stable and backward compatible
- OAuth providers will maintain service availability
- Users will have basic understanding of cryptocurrency trading concepts

### 6.2 Dependencies
- PostgreSQL database with TimescaleDB extension
- OAuth provider services (Google, WeChat)
- Cryptocurrency trading platform APIs
- Docker runtime environment

## 7. Resources

### 7.1 Domain Configuration
- **Production Domain**: tiris.ai
- **Development Domain**: dev.tiris.ai
- **API Base URL**: https://api.tiris.ai/v1
- **Development API URL**: https://api.dev.tiris.ai/v1
- **Portal URL**: https://tiris.ai
- **Development Portal URL**: https://dev.tiris.ai

### 7.2 SSL/TLS Requirements
- All domains MUST use HTTPS with valid SSL certificates
- API endpoints MUST enforce TLS 1.2 minimum
- Development environment SHOULD use valid certificates (not self-signed)

## 8. Future Considerations

### 8.1 Planned Enhancements
- Additional OAuth providers
- Support for more cryptocurrency trading platforms
- Advanced analytics and reporting features
- Real-time notification system
- Multi-language support

### 8.2 Scalability Planning
- Kubernetes deployment for production scaling
- Database sharding strategies for large user bases
- Caching layer implementation for frequently accessed data
- API rate limiting and throttling mechanisms