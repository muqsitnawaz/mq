# Configuration Reference

## Overview

Engram can be configured through multiple methods, with the following precedence order:

1. Command-line flags (highest priority)
2. Environment variables
3. Configuration files
4. Default values (lowest priority)

## Configuration File

### Location

The main configuration file is located at:

- **Linux/macOS:** `~/.engram/config.yaml`
- **Windows:** `%USERPROFILE%\.engram\config.yaml`
- **Custom:** Use `--config` flag to specify location

### Basic Configuration

```yaml
# ~/.engram/config.yaml

# General settings
version: 1.0.0
environment: production

# Cache configuration
cache:
  enabled: true
  size: 100MB
  ttl: 24h
  directory: ~/.engram/cache
  compression: true

# File watching
watch:
  enabled: true
  interval: 1s
  patterns:
    - "**/*.md"
    - "**/*.markdown"
  ignore:
    - "**/node_modules/**"
    - "**/.git/**"
    - "**/vendor/**"

# Logging
logging:
  level: info
  format: json
  output: stderr
  file: ~/.engram/logs/mq.log
  rotate:
    enabled: true
    max_size: 100MB
    max_age: 30d
    max_backups: 10
```

### Advanced Configuration

```yaml
# Performance tuning
performance:
  parser:
    max_file_size: 50MB
    timeout: 30s
    concurrent_files: 10

  cache:
    strategy: lru
    preload:
      enabled: true
      patterns:
        - "docs/**/*.md"
        - "README.md"

  indexing:
    enabled: true
    rebuild_interval: 1h
    algorithms:
      - btree
      - fulltext
      - semantic

# Security settings
security:
  api:
    rate_limit: 1000
    rate_window: 1m
    max_request_size: 10MB

  auth:
    enabled: false
    providers:
      - jwt
      - api_key

  encryption:
    at_rest: true
    algorithm: AES-256-GCM
    key_rotation: 30d
```

## Environment Variables

All configuration options can be set via environment variables:

```bash
# Format: ENGRAM_<SECTION>_<KEY>
export ENGRAM_CACHE_ENABLED=true
export ENGRAM_CACHE_SIZE=200MB
export ENGRAM_CACHE_TTL=48h

# Nested values use double underscore
export ENGRAM_PERFORMANCE_PARSER_MAX_FILE_SIZE=100MB
export ENGRAM_SECURITY_API_RATE_LIMIT=2000

# Arrays use comma separation
export ENGRAM_WATCH_PATTERNS="*.md,*.markdown,*.mdown"
export ENGRAM_WATCH_IGNORE="node_modules,vendor,.git"
```

## Semantic Search Configuration

### OpenAI Provider

```yaml
embeddings:
  provider: openai
  api_key: ${OPENAI_API_KEY}  # Environment variable reference
  model: text-embedding-ada-002

  options:
    batch_size: 100
    retry:
      enabled: true
      max_attempts: 3
      backoff: exponential

    cache:
      enabled: true
      ttl: 7d

    rate_limit:
      requests_per_minute: 3000
      tokens_per_minute: 1000000
```

### Local Embeddings

```yaml
embeddings:
  provider: local
  model: sentence-transformers/all-MiniLM-L6-v2

  options:
    device: cuda  # or cpu
    batch_size: 32
    max_seq_length: 512

    model_cache:
      directory: ~/.engram/models
      auto_download: true
```

### HuggingFace Provider

```yaml
embeddings:
  provider: huggingface
  api_key: ${HF_API_KEY}
  model: sentence-transformers/all-mpnet-base-v2
  endpoint: https://api-inference.huggingface.co/models

  options:
    timeout: 30s
    use_gpu: true
    precision: float16
```

## Database Configuration

### PostgreSQL

```yaml
database:
  type: postgresql
  connection:
    host: ${DB_HOST:-localhost}
    port: ${DB_PORT:-5432}
    database: ${DB_NAME:-engram}
    username: ${DB_USER:-engram_user}
    password: ${DB_PASSWORD}

    # Connection pool settings
    pool:
      min_connections: 2
      max_connections: 20
      max_idle_time: 5m
      connection_timeout: 10s

    # SSL configuration
    ssl:
      enabled: true
      mode: require  # disable, allow, prefer, require, verify-ca, verify-full
      cert: /path/to/client-cert.pem
      key: /path/to/client-key.pem
      root_cert: /path/to/ca-cert.pem
```

### SQLite (Development)

```yaml
database:
  type: sqlite
  connection:
    path: ./engram.db

    # SQLite specific options
    options:
      journal_mode: WAL
      synchronous: NORMAL
      cache_size: -64000  # 64MB
      foreign_keys: true
      busy_timeout: 5000
```

### MongoDB

```yaml
database:
  type: mongodb
  connection:
    uri: ${MONGODB_URI:-mongodb://localhost:27017}
    database: engram

    options:
      auth_source: admin
      replica_set: rs0
      read_preference: primaryPreferred
      write_concern:
        w: majority
        j: true
        wtimeout: 5000
```

## Server Configuration

### HTTP Server

```yaml
server:
  http:
    enabled: true
    host: 0.0.0.0
    port: 8080

    # TLS configuration
    tls:
      enabled: false
      cert: /path/to/server.crt
      key: /path/to/server.key

    # CORS settings
    cors:
      enabled: true
      allowed_origins:
        - http://localhost:3000
        - https://app.example.com
      allowed_methods:
        - GET
        - POST
        - OPTIONS
      allowed_headers:
        - Content-Type
        - Authorization
      expose_headers:
        - X-Request-ID
      max_age: 86400

    # Request handling
    timeouts:
      read: 30s
      write: 30s
      idle: 120s
      shutdown: 30s

    limits:
      max_request_size: 10MB
      max_header_size: 1MB
      max_connections: 1000
```

### gRPC Server

```yaml
server:
  grpc:
    enabled: true
    port: 9090

    # TLS for gRPC
    tls:
      enabled: true
      cert: /path/to/server.crt
      key: /path/to/server.key
      client_auth: require  # none, request, require
      client_ca: /path/to/ca.crt

    # gRPC specific options
    options:
      max_receive_message_size: 10MB
      max_send_message_size: 10MB
      keepalive:
        time: 30s
        timeout: 10s
        permit_without_stream: true
```

## Query Engine Configuration

```toml
# Alternative TOML configuration example

[query]
default_limit = 100
max_limit = 1000
timeout = "30s"
cache_results = true

[query.optimizer]
enabled = true
rules = [
    "constant_folding",
    "predicate_pushdown",
    "dead_code_elimination",
    "common_subexpression_elimination"
]

[query.planner]
cost_based = true
statistics_sample_rate = 0.1
join_reorder = true
parallel_execution = true
max_parallel_workers = 4

[query.execution]
memory_limit = "1GB"
temp_directory = "/tmp/engram"
spill_to_disk = true
vectorized = true
```

## Output Formats Configuration

```json
{
  "output": {
    "formats": {
      "json": {
        "enabled": true,
        "indent": 2,
        "escape_html": false,
        "sort_keys": false,
        "compact": false
      },
      "yaml": {
        "enabled": true,
        "indent": 2,
        "line_width": 80,
        "explicit_start": false,
        "explicit_end": false
      },
      "xml": {
        "enabled": true,
        "indent": 2,
        "root_element": "result",
        "include_declaration": true,
        "encoding": "UTF-8"
      },
      "csv": {
        "enabled": true,
        "delimiter": ",",
        "quote_char": "\"",
        "escape_char": "\\",
        "include_header": true,
        "line_ending": "\\n"
      }
    },
    "default": "json",
    "pretty_print": true,
    "color_output": true,
    "terminal_width": 120
  }
}
```

## Plugin Configuration

```yaml
plugins:
  enabled: true
  directory: ~/.engram/plugins

  # Auto-load plugins
  autoload:
    - name: syntax-highlighter
      enabled: true
      config:
        theme: monokai
        line_numbers: true

    - name: markdown-extensions
      enabled: true
      config:
        extensions:
          - tables
          - footnotes
          - definition_lists
          - abbreviations

    - name: custom-operators
      enabled: true
      source: https://github.com/user/mq-custom-ops
      version: v1.0.0

  # Security settings for plugins
  security:
    sandbox: true
    network_access: false
    filesystem_access: read
    max_memory: 100MB
    timeout: 10s
```

## Monitoring & Metrics

```yaml
monitoring:
  metrics:
    enabled: true
    provider: prometheus
    endpoint: /metrics
    port: 9091

    collectors:
      - go_collector
      - process_collector
      - custom_collector

    labels:
      environment: production
      service: engram
      version: 1.0.0

  tracing:
    enabled: true
    provider: jaeger

    jaeger:
      agent_endpoint: localhost:6831
      collector_endpoint: http://localhost:14268/api/traces
      service_name: engram-mq

      sampling:
        type: adaptive
        max_traces_per_second: 100
        initial_sampling_rate: 0.001

  health:
    enabled: true
    endpoint: /health
    checks:
      - database
      - cache
      - filesystem
```

## Profiles

Different configuration profiles for various environments:

```yaml
profiles:
  development:
    cache:
      enabled: false
    logging:
      level: debug
      format: text
    database:
      type: sqlite
      connection:
        path: ./dev.db

  testing:
    cache:
      enabled: true
      size: 10MB
    logging:
      level: warn
    database:
      type: sqlite
      connection:
        path: :memory:

  production:
    cache:
      enabled: true
      size: 1GB
    logging:
      level: info
      format: json
    database:
      type: postgresql
    monitoring:
      metrics:
        enabled: true
      tracing:
        enabled: true

# Active profile selection
active_profile: ${ENGRAM_PROFILE:-development}
```

## Command-Line Overrides

Override any configuration via command-line flags:

```bash
# Override cache settings
mq --cache.size=200MB --cache.ttl=48h query '.headings' doc.md

# Override database connection
mq --database.type=sqlite --database.connection.path=./test.db load *.md

# Override logging
mq --logging.level=debug --logging.format=text server

# Use different configuration file
mq --config=/etc/engram/production.yaml query '.sections' doc.md

# Disable specific features
mq --cache.enabled=false --watch.enabled=false query '.code' doc.md
```

## Validation & Schema

Configuration validation rules:

```yaml
schema:
  version: 1.0.0

  validation:
    cache:
      size:
        type: string
        pattern: '^[0-9]+(B|KB|MB|GB)$'
        required: false
        default: "100MB"

      ttl:
        type: duration
        minimum: 1s
        maximum: 168h
        required: false
        default: "24h"

    database:
      type:
        type: enum
        values: [postgresql, sqlite, mongodb, mysql]
        required: true

      connection:
        type: object
        required: true
        properties:
          host:
            type: string
            format: hostname
          port:
            type: integer
            minimum: 1
            maximum: 65535
```

## Migration from Older Versions

```yaml
# Legacy configuration migration
migration:
  from_version: 0.9.0

  mappings:
    # Old path -> New path
    "cache.max_size": "cache.size"
    "cache.expire_after": "cache.ttl"
    "db.type": "database.type"
    "db.conn_string": "database.connection.uri"

  deprecated:
    - "cache.algorithm"  # Now auto-selected
    - "query.legacy_mode"  # Removed in v1.0

  warnings:
    - field: "security.api.auth_enabled"
      message: "Use security.auth.enabled instead"
    - field: "embeddings.batch_limit"
      message: "Use embeddings.options.batch_size instead"
```

---

*Configuration Reference Version: 1.0.0 | Last Updated: January 2024*