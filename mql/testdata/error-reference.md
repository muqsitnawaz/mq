# Error Reference

## Error Categories

### Syntax Errors (1000-1999)

#### ERR-1001: Invalid Query Syntax

**Error Message:** `Syntax error at position {position}: unexpected token '{token}'`

**Causes:**
- Missing closing parenthesis or bracket
- Invalid operator usage
- Malformed string literals

**Example:**
```bash
# Wrong
mq '.section("API"' document.md
#              ^ Missing closing parenthesis

# Correct
mq '.section("API")' document.md
```

**Solution:**
1. Check for matching parentheses and brackets
2. Ensure strings are properly quoted
3. Verify operator syntax

---

#### ERR-1002: Unknown Selector

**Error Message:** `Unknown selector: {selector}`

**Causes:**
- Typo in selector name
- Using deprecated or non-existent selector
- Case sensitivity issue

**Example:**
```bash
# Wrong
mq '.headers' document.md  # Should be .headings

# Correct
mq '.headings' document.md
```

**Solution:**
- Consult the selector reference
- Check spelling and case
- Use `mq --list-selectors` to see available selectors

---

#### ERR-1003: Invalid Regular Expression

**Error Message:** `Invalid regex pattern: {pattern} - {error}`

**Causes:**
- Unescaped special characters
- Invalid regex syntax
- Unclosed groups

**Example:**
```bash
# Wrong
mq '.heading(/API [Methods/)' document.md
#                  ^ Unclosed bracket

# Correct
mq '.heading(/API \[Methods\]/)' document.md
```

**Solution:**
1. Escape special regex characters
2. Test regex patterns separately
3. Use online regex validators

### Runtime Errors (2000-2999)

#### ERR-2001: File Not Found

**Error Message:** `Cannot read file: {filepath} - No such file or directory`

**Causes:**
- Incorrect file path
- File doesn't exist
- Permission issues

**Example:**
```bash
# Check if file exists
ls -la document.md

# Use absolute path if needed
mq '.headings' /absolute/path/to/document.md

# Check permissions
chmod +r document.md
```

**Solution:**
- Verify file path and existence
- Check file permissions
- Use relative or absolute paths correctly

---

#### ERR-2002: Memory Limit Exceeded

**Error Message:** `Query execution exceeded memory limit: {used}MB > {limit}MB`

**Causes:**
- Processing very large files
- Complex queries on multiple files
- Insufficient system memory

**Example:**
```yaml
# Increase memory limit in config
query:
  execution:
    memory_limit: "2GB"  # Increase from default
```

**Solution:**
1. Increase memory limit in configuration
2. Use chunking for large files
3. Optimize query to be more selective
4. Process files individually

---

#### ERR-2003: Timeout Error

**Error Message:** `Query execution timeout after {duration}s`

**Causes:**
- Complex query on large dataset
- Slow I/O operations
- Deadlock in processing

**Example:**
```bash
# Increase timeout
mq --timeout=60s '.complex_query' large-file.md

# Or in configuration
query:
  timeout: "60s"
```

**Solution:**
- Increase timeout value
- Optimize query performance
- Use caching for repeated queries
- Break complex queries into steps

### Type Errors (3000-3999)

#### ERR-3001: Type Mismatch

**Error Message:** `Type mismatch: expected {expected}, got {actual}`

**Causes:**
- Applying wrong operation to data type
- Invalid type conversion
- Null or undefined values

**Example:**
```bash
# Wrong - trying to get text from a number
mq '.headings | length | .text' document.md

# Correct - length returns a number
mq '.headings | length' document.md
```

**Solution:**
1. Check data types in pipeline
2. Use appropriate operations for each type
3. Handle null values with conditionals

---

#### ERR-3002: Invalid Argument Type

**Error Message:** `Function {function} expects {expected} argument, got {actual}`

**Causes:**
- Wrong argument type passed to function
- Missing required arguments
- Too many arguments

**Example:**
```bash
# Wrong - select expects boolean expression
mq '.headings | select("level 2")' document.md

# Correct
mq '.headings | select(.level == 2)' document.md
```

**Solution:**
- Check function signature
- Verify argument types
- Use correct syntax for conditions

### I/O Errors (4000-4999)

#### ERR-4001: Permission Denied

**Error Message:** `Permission denied: cannot access {resource}`

**Causes:**
- Insufficient file permissions
- Protected system directories
- SELinux or AppArmor restrictions

**Solution:**
```bash
# Check permissions
ls -la file.md

# Fix permissions
chmod 644 file.md

# Check SELinux context (Linux)
ls -Z file.md

# Run with appropriate user
sudo -u appropriate_user mq '.headings' file.md
```

---

#### ERR-4002: Disk Full

**Error Message:** `Cannot write to cache: No space left on device`

**Causes:**
- Insufficient disk space
- Cache directory full
- Temp directory full

**Solution:**
```bash
# Check disk space
df -h

# Clear cache
mq cache clear

# Change cache location
export ENGRAM_CACHE_DIRECTORY=/path/with/space

# Disable cache temporarily
mq --cache.enabled=false '.query' file.md
```

---

#### ERR-4003: Network Error

**Error Message:** `Network error: {details}`

**Causes:**
- API endpoint unreachable
- Internet connection issues
- Firewall blocking requests

**Troubleshooting:**
```bash
# Test connectivity
ping api.engram.ai
curl -I https://api.engram.ai/health

# Check proxy settings
echo $HTTP_PROXY
echo $HTTPS_PROXY

# Use offline mode
mq --offline '.query' file.md
```

### API Errors (5000-5999)

#### ERR-5001: Authentication Failed

**Error Message:** `Authentication failed: Invalid API key`

**Causes:**
- Invalid or expired API key
- Missing authentication headers
- Wrong authentication method

**Solution:**
```bash
# Set API key
export ENGRAM_API_KEY="your-valid-key"

# Or in config
embeddings:
  api_key: "your-valid-key"

# Verify key is set
echo $ENGRAM_API_KEY
```

---

#### ERR-5002: Rate Limit Exceeded

**Error Message:** `Rate limit exceeded: {limit} requests per {window}`

**Causes:**
- Too many requests in time window
- Burst limit exceeded
- Account quota reached

**Solution:**
1. Implement exponential backoff
2. Upgrade to higher tier
3. Cache results to reduce API calls
4. Batch operations when possible

```python
import time
import random

def retry_with_backoff(func, max_retries=5):
    for i in range(max_retries):
        try:
            return func()
        except RateLimitError:
            wait_time = (2 ** i) + random.uniform(0, 1)
            time.sleep(wait_time)
    raise Exception("Max retries exceeded")
```

---

#### ERR-5003: Invalid API Response

**Error Message:** `Invalid response from API: {details}`

**Causes:**
- API version mismatch
- Corrupted response data
- API service issues

**Solution:**
- Check API status page
- Verify API version compatibility
- Report issue if persistent
- Use fallback mechanism

### Configuration Errors (6000-6999)

#### ERR-6001: Invalid Configuration

**Error Message:** `Configuration error: {field} - {details}`

**Causes:**
- YAML/JSON syntax errors
- Invalid configuration values
- Missing required fields

**Example of common mistakes:**
```yaml
# Wrong - invalid YAML indentation
cache:
enabled: true  # Should be indented

# Correct
cache:
  enabled: true

# Wrong - invalid value type
cache:
  size: 100  # Should be string with unit

# Correct
cache:
  size: "100MB"
```

**Solution:**
1. Validate YAML/JSON syntax
2. Check configuration schema
3. Use `mq config validate` command

---

#### ERR-6002: Conflicting Configuration

**Error Message:** `Configuration conflict: {option1} cannot be used with {option2}`

**Causes:**
- Mutually exclusive options
- Incompatible settings
- Override conflicts

**Solution:**
- Review configuration priorities
- Remove conflicting options
- Use profiles for different environments

### Cache Errors (7000-7999)

#### ERR-7001: Cache Corruption

**Error Message:** `Cache corruption detected for: {file}`

**Causes:**
- Incomplete write operation
- File system errors
- Version mismatch

**Solution:**
```bash
# Clear specific file from cache
mq cache remove file.md

# Clear entire cache
mq cache clear

# Rebuild cache
mq cache rebuild

# Verify cache integrity
mq cache verify
```

---

#### ERR-7002: Cache Version Mismatch

**Error Message:** `Cache version {cache_version} incompatible with mq version {mq_version}`

**Causes:**
- MQ upgrade with old cache
- Downgrade attempt
- Mixed version deployment

**Solution:**
1. Clear old cache: `mq cache clear`
2. Rebuild with current version: `mq cache rebuild`
3. Update all instances to same version

### Semantic Search Errors (8000-8999)

#### ERR-8001: Embedding Generation Failed

**Error Message:** `Failed to generate embeddings: {details}`

**Causes:**
- API service unavailable
- Invalid input text
- Model loading failure

**Solution:**
```yaml
# Fallback to different provider
embeddings:
  provider: local  # Instead of OpenAI
  model: all-MiniLM-L6-v2

# Or disable semantic features
embeddings:
  enabled: false
```

---

#### ERR-8002: Vector Database Error

**Error Message:** `Vector database operation failed: {operation}`

**Causes:**
- Database connection issues
- Index corruption
- Dimension mismatch

**Solution:**
1. Rebuild vector index: `mq index rebuild --vectors`
2. Check embedding dimensions match
3. Verify database connectivity
4. Clear and regenerate embeddings

## Error Recovery Strategies

### Automatic Recovery

```yaml
# Configure automatic recovery
recovery:
  auto_retry:
    enabled: true
    max_attempts: 3
    backoff: exponential

  fallback:
    enabled: true
    strategies:
      - cache_only
      - basic_search
      - error_message

  circuit_breaker:
    enabled: true
    threshold: 5
    timeout: 60s
```

### Manual Recovery Steps

1. **For Persistent Errors:**
   ```bash
   # Reset to defaults
   mq reset

   # Validate installation
   mq doctor

   # Run in debug mode
   mq --debug '.query' file.md 2> debug.log
   ```

2. **For Data Corruption:**
   ```bash
   # Backup current state
   cp -r ~/.engram ~/.engram.backup

   # Clear all caches and indexes
   mq cache clear
   mq index clear

   # Reinitialize
   mq init --force
   ```

3. **For Performance Issues:**
   ```bash
   # Profile query execution
   mq --profile '.complex_query' file.md

   # Analyze bottlenecks
   mq analyze performance.prof

   # Optimize query
   mq optimize '.complex_query'
   ```

## Getting Help

### Debug Information Collection

When reporting errors, include:

```bash
# Version information
mq --version

# System information
mq doctor --verbose

# Configuration dump
mq config dump

# Debug log with trace
mq --log-level=trace '.failing_query' file.md 2> trace.log

# Environment variables
env | grep ENGRAM
```

### Support Channels

- **GitHub Issues:** [github.com/engram/engram/issues](https://github.com/engram/engram/issues)
- **Discord:** [discord.gg/engram](https://discord.gg/engram)
- **Email:** support@engram.ai
- **Stack Overflow:** Tag with `engram-mq`

---

*Error Reference Version: 1.0.0 | Last Updated: January 2024*