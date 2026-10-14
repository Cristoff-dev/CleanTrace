# 🛡️ CleanTrace: High-Performance Log Sanitizer & PII Masking Pipeline

![Go Version](https://img.shields.io/badge/Go-1.27.2-00ADD8?style=flat&logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-JSONB-4169E1?style=flat&logo=postgresql)
![Docker](https://img.shields.io/badge/Docker-Multi--Stage-2496ED?style=flat&logo=docker)
![Compliance](https://img.shields.io/badge/Compliance-HIPAA%20%7C%20PCI--DSS%20%7C%20GDPR-success)

CleanTrace is an enterprise-grade middleware microservice designed to intercept, sanitize, and normalize application telemetry before it persists to storage. Engineered for healthcare, fintech, and modern e-commerce architectures, it acts as a zero-trust buffer to prevent accidental leakage of Personally Identifiable Information (PII) into company logs.

## 💡 The Business Value

Logging a customer's plain-text password, credit card number, or JWT session token can expose a company to severe legal liabilities and million-dollar audit fines. CleanTrace intercepts high-throughput log streams in microseconds, automatically redacting sensitive patterns before they hit the disk, ensuring full HIPAA and PCI-DSS compliance.

## 🚀 Engineering Highlights & Architecture

* **Anti-ReDoS Masking Engine (RE2):** Leverages Go's native RE2 regular expression engine. Unlike backtracking engines found in Python or Node.js, RE2 guarantees linear-time execution O(n), rendering the system immune to Regular Expression Denial of Service (ReDoS) attacks during traffic spikes.
* **Bounded Concurrency (Semaphore Pattern):** Log arrays are parsed and sanitized concurrently utilizing lightweight goroutines. A semaphore strictly bounds maximum active workers to hardware threads, preventing Go scheduler starvation under heavy I/O loads.
* **Unstructured Payload Normalization:** Validates incoming payloads on the fly. Plain-text logs are automatically wrapped into valid JSON structures prior to database insertion, guaranteeing schema integrity.
* **O(1) Bulk Persistence:** Interacts with PostgreSQL asynchronously using `pgxpool`. Instead of row-by-row insert loops, CleanTrace packages arrays of processed logs into a single TCP network packet (`pgx.Batch`), drastically minimizing database connection overhead.
* **Hexagonal Architecture:** Strictly separates Domain models, Ports (Interfaces), and Adapters (HTTP Ingestion, PostgreSQL Storage), ensuring the system is highly testable and decoupled from underlying infrastructure.

## 📁 Repository Structure

    cleantrace/
    ├── cmd/cleantrace/           # Application entrypoint & Wire-up
    ├── internal/
    │   ├── core/domain/          # Core business entities
    │   ├── core/ports/           # Primary & Secondary Interfaces
    │   ├── sanitizer/            # RE2 Regex PII Masking Engine
    │   ├── ingestion/            # Concurrent HTTP Handler
    │   └── storage/              # PostgreSQL pgx.Batch Storage Adapter
    ├── pkg/logger/               # Zero-allocation Structured Logging (Zap)
    ├── Dockerfile                # Multi-stage Distroless build
    └── docker-compose.yml        # Isolated local infrastructure

## 🛠️ Quick Start

### 1. Start Infrastructure
Spin up the PostgreSQL database using Docker Compose:

    docker-compose up -d

### 2. Initialize Database Schema
Connect to PostgreSQL and create the optimized JSONB table:

    CREATE TABLE sanitized_logs (
        id VARCHAR(255) PRIMARY KEY,
        timestamp TIMESTAMPTZ NOT NULL,
        environment VARCHAR(50) NOT NULL,
        level VARCHAR(20) NOT NULL,
        service VARCHAR(100) NOT NULL,
        raw_payload JSONB NOT NULL,
        masked_data BOOLEAN NOT NULL
    );
    CREATE INDEX idx_raw_payload_gin ON sanitized_logs USING GIN (raw_payload);

### 3. Run the Application
Start the Go server:

    go run ./cmd/cleantrace

### 4. Dispatch a Test Payload
Simulate a log stream containing a leaked credit card and an email:

    curl -X POST http://localhost:8080/api/v1/logs/collect \
    -H "Content-Type: application/json" \
    -d '{
      "logs": [
        {
          "id": "log-001",
          "environment": "production",
          "level": "ERROR",
          "service": "payment-service",
          "raw_payload": "{\"user\":\"john.doe@email.com\",\"card\":\"4111-1111-1111-1111\"}"
        }
      ]
    }'

## 🛡️ Production Standards

* **Distroless Containerization:** Built with a multi-stage Dockerfile running on `gcr.io/distroless/static:nonroot`, eliminating shell access and heavily reducing the attack surface.
* **Graceful Shutdown:** Traps system signals (SIGINT, SIGTERM) to cleanly flush active database batches and terminate HTTP connections before pod destruction.
