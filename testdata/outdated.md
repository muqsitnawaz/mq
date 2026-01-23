# Legacy API Documentation

**WARNING: This documentation is outdated. See [new docs](./api-v2.md).**

## Deprecated Endpoints

### GET /api/v1/users (DEPRECATED)

Use `/api/v2/users` instead.

```bash
# Old way (don't use)
curl https://api.example.com/v1/users

# New way
curl https://api.example.com/v2/users
```

## Removed Features

The following features have been removed:

- XML response format (removed in v2.0)
- Basic auth (removed in v1.5)
- Sync API (removed in v2.0)

## Migration Guide

TODO: Add migration guide

## Old Configuration

```json
{
  "api_version": "v1",
  "format": "xml",
  "auth": "basic"
}
```

This config format is no longer supported.

## Broken References

- [Old Dashboard](https://old.example.com/dashboard) - no longer exists
- [Legacy Docs](https://docs.example.com/v1) - redirects to v2
- [Sunset Blog Post](https://blog.example.com/2023/api-v1-sunset)
