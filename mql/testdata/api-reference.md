# API Reference

## Overview

The Engram API provides a comprehensive set of endpoints for managing AI agent memory systems. This document covers authentication, available endpoints, request/response formats, and code examples.

### Base URL

```
https://api.engram.ai/v1
```

### Version Information

Current API Version: **v1.0.0**
Release Date: 2024-01-15
Status: **Stable**

## Authentication

### API Keys

All API requests must include an authentication token in the request headers. You can generate API keys from your dashboard.

```python
# Python authentication example
import requests

headers = {
    'Authorization': 'Bearer YOUR_API_KEY',
    'Content-Type': 'application/json'
}

response = requests.get('https://api.engram.ai/v1/memories', headers=headers)
```

```javascript
// JavaScript authentication example
const axios = require('axios');

const client = axios.create({
    baseURL: 'https://api.engram.ai/v1',
    headers: {
        'Authorization': 'Bearer YOUR_API_KEY',
        'Content-Type': 'application/json'
    }
});
```

```go
// Go authentication example
package main

import (
    "net/http"
)

func main() {
    client := &http.Client{}
    req, _ := http.NewRequest("GET", "https://api.engram.ai/v1/memories", nil)
    req.Header.Add("Authorization", "Bearer YOUR_API_KEY")
    req.Header.Add("Content-Type", "application/json")

    resp, _ := client.Do(req)
    defer resp.Body.Close()
}
```

### OAuth 2.0

For applications requiring user authentication, we support OAuth 2.0 flow.

#### Authorization URL
```
https://auth.engram.ai/oauth/authorize
```

#### Token Exchange

```curl
curl -X POST https://auth.engram.ai/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code" \
  -d "code=AUTH_CODE" \
  -d "client_id=YOUR_CLIENT_ID" \
  -d "client_secret=YOUR_CLIENT_SECRET" \
  -d "redirect_uri=YOUR_REDIRECT_URI"
```

## Endpoints

### Memory Operations

#### Create Memory

Creates a new memory entry in the system.

**Endpoint:** `POST /memories`

**Request Body:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| content | string | Yes | The memory content in markdown format |
| metadata | object | No | Additional metadata for the memory |
| tags | array | No | Array of tags for categorization |
| priority | integer | No | Priority level (1-10) |
| owner | string | Yes | Owner identifier |

**Example Request:**

```python
# Python example for creating memory
import json
import requests

data = {
    "content": "# Project Architecture\n\n## Overview\nThe system uses microservices...",
    "metadata": {
        "project": "engram",
        "version": "1.0.0"
    },
    "tags": ["architecture", "documentation"],
    "priority": 8,
    "owner": "agent-001"
}

response = requests.post(
    'https://api.engram.ai/v1/memories',
    headers=headers,
    data=json.dumps(data)
)

print(response.json())
```

**Response:**

```json
{
    "id": "mem_abc123xyz",
    "content": "# Project Architecture\n\n## Overview\nThe system uses microservices...",
    "metadata": {
        "project": "engram",
        "version": "1.0.0",
        "created_at": "2024-01-20T10:30:00Z",
        "updated_at": "2024-01-20T10:30:00Z"
    },
    "tags": ["architecture", "documentation"],
    "priority": 8,
    "owner": "agent-001",
    "status": "active"
}
```

#### List Memories

Retrieves a list of memories with optional filtering.

**Endpoint:** `GET /memories`

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| owner | string | Filter by owner |
| tags | string | Comma-separated list of tags |
| priority_min | integer | Minimum priority level |
| priority_max | integer | Maximum priority level |
| limit | integer | Number of results (max 100) |
| offset | integer | Pagination offset |

```javascript
// JavaScript example for listing memories
const params = {
    owner: 'agent-001',
    tags: 'architecture,documentation',
    priority_min: 5,
    limit: 20
};

client.get('/memories', { params })
    .then(response => {
        console.log(response.data);
    })
    .catch(error => {
        console.error('Error:', error);
    });
```

#### Update Memory

Updates an existing memory entry.

**Endpoint:** `PATCH /memories/{id}`

```go
// Go example for updating memory
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

func updateMemory(id string, updates map[string]interface{}) error {
    jsonData, _ := json.Marshal(updates)

    req, _ := http.NewRequest(
        "PATCH",
        fmt.Sprintf("https://api.engram.ai/v1/memories/%s", id),
        bytes.NewBuffer(jsonData),
    )

    req.Header.Set("Authorization", "Bearer YOUR_API_KEY")
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    return nil
}
```

#### Delete Memory

Permanently deletes a memory entry.

**Endpoint:** `DELETE /memories/{id}`

```curl
curl -X DELETE https://api.engram.ai/v1/memories/mem_abc123xyz \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Query Operations

#### Execute Query

Runs an mq query against stored memories.

**Endpoint:** `POST /query`

**Request Body:**

```json
{
    "query": ".section('Architecture') | .code('python') | .with_context",
    "scope": ["mem_abc123", "mem_def456"],
    "options": {
        "limit": 10,
        "semantic_search": true
    }
}
```

**Example with Different Languages:**

```python
# Python query execution
query_data = {
    "query": ".headings | select(.level <= 2) | .text",
    "options": {
        "limit": 50
    }
}

results = requests.post(
    'https://api.engram.ai/v1/query',
    headers=headers,
    json=query_data
).json()
```

```javascript
// JavaScript query execution
async function executeQuery(query) {
    const response = await client.post('/query', {
        query: query,
        options: {
            semantic_search: true,
            chunk_size: 4000
        }
    });

    return response.data.results;
}

// Usage
const results = await executeQuery(".section('API') | .code | .with_context");
```

### Batch Operations

#### Bulk Import

Import multiple memories in a single request.

**Endpoint:** `POST /memories/bulk`

**Request Format:**

```json
{
    "memories": [
        {
            "content": "# Memory 1\nContent here...",
            "tags": ["tag1", "tag2"],
            "owner": "agent-001"
        },
        {
            "content": "# Memory 2\nMore content...",
            "tags": ["tag3"],
            "owner": "agent-002"
        }
    ],
    "options": {
        "validate": true,
        "on_conflict": "update"
    }
}
```

## Error Handling

### Error Response Format

All errors follow a consistent format:

```json
{
    "error": {
        "code": "VALIDATION_ERROR",
        "message": "Invalid query syntax",
        "details": {
            "field": "query",
            "reason": "Unclosed parenthesis at position 25"
        },
        "timestamp": "2024-01-20T10:30:00Z",
        "request_id": "req_xyz789"
    }
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| AUTHENTICATION_FAILED | 401 | Invalid or missing API key |
| INSUFFICIENT_PERMISSIONS | 403 | Operation not allowed for user |
| RESOURCE_NOT_FOUND | 404 | Memory or resource not found |
| VALIDATION_ERROR | 400 | Request validation failed |
| RATE_LIMIT_EXCEEDED | 429 | Too many requests |
| INTERNAL_ERROR | 500 | Server error |

### Rate Limiting

API requests are rate-limited per API key:

- **Standard Tier:** 100 requests per minute
- **Pro Tier:** 1000 requests per minute
- **Enterprise Tier:** Custom limits

Rate limit information is included in response headers:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1642680000
```

## Webhooks

### Webhook Configuration

Configure webhooks to receive real-time updates about memory changes.

**Endpoint:** `POST /webhooks`

```json
{
    "url": "https://your-app.com/webhook",
    "events": ["memory.created", "memory.updated", "memory.deleted"],
    "secret": "your_webhook_secret"
}
```

### Webhook Payload

```json
{
    "event": "memory.updated",
    "timestamp": "2024-01-20T10:30:00Z",
    "data": {
        "memory_id": "mem_abc123xyz",
        "changes": {
            "tags": {
                "added": ["new-tag"],
                "removed": ["old-tag"]
            }
        }
    },
    "signature": "sha256=abcdef123456..."
}
```

## SDK Examples

### Python SDK

```python
from engram import EngramClient

client = EngramClient(api_key="YOUR_API_KEY")

# Create memory
memory = client.memories.create(
    content="# Important Note\n\nThis is a test memory.",
    tags=["test", "example"],
    priority=5
)

# Query memories
results = client.query(
    ".section('Important') | .text",
    limit=10
)

# Update memory
client.memories.update(
    memory.id,
    tags=["test", "updated"]
)
```

### JavaScript/TypeScript SDK

```typescript
import { EngramClient } from '@engram/sdk';

const client = new EngramClient({
    apiKey: process.env.ENGRAM_API_KEY
});

// Async operations
async function manageMemories() {
    // Create
    const memory = await client.memories.create({
        content: '# Documentation\n\nAPI usage guide...',
        metadata: { version: '1.0' }
    });

    // Query
    const results = await client.query({
        query: '.headings | .text',
        options: { limit: 20 }
    });

    // Delete
    await client.memories.delete(memory.id);
}
```

## Best Practices

### Query Optimization

1. Use specific selectors to reduce processing time
2. Leverage caching for frequently accessed memories
3. Batch operations when possible
4. Use semantic search for natural language queries

### Security Recommendations

- Rotate API keys regularly
- Use environment variables for sensitive data
- Implement request signing for webhooks
- Validate all input data

### Performance Tips

```python
# Good: Specific query with filters
results = client.query(
    ".section('API') | .code('python') | select(.lines > 10)",
    limit=20
)

# Bad: Overly broad query
results = client.query(
    ".descendants | .text",
    limit=1000
)
```

## Migration Guide

### Migrating from v0.x to v1.0

Major changes in v1.0:

1. **Authentication**: Bearer tokens instead of API keys in query params
2. **Endpoints**: RESTful naming conventions
3. **Query Language**: Enhanced mq syntax with new operators

```python
# Old (v0.x)
response = requests.get(
    'https://api.engram.ai/get_memories?api_key=KEY&owner=agent'
)

# New (v1.0)
response = requests.get(
    'https://api.engram.ai/v1/memories',
    headers={'Authorization': 'Bearer KEY'},
    params={'owner': 'agent'}
)
```

## Support

For API support:
- Email: api-support@engram.ai
- Documentation: https://docs.engram.ai
- Status Page: https://status.engram.ai
- Community Forum: https://community.engram.ai

---

*Last updated: January 20, 2024*