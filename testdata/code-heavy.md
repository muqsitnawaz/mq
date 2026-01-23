# Code Examples

Various code snippets in different languages.

## Python Examples

### Basic Script

```python
def hello():
    print("Hello, World!")

if __name__ == "__main__":
    hello()
```

### Async Code

```python
import asyncio

async def fetch_data():
    await asyncio.sleep(1)
    return {"status": "ok"}

async def main():
    result = await fetch_data()
    print(result)
```

## Go Examples

### HTTP Server

```go
package main

import (
    "fmt"
    "net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello, World!")
}

func main() {
    http.HandleFunc("/", handler)
    http.ListenAndServe(":8080", nil)
}
```

### Concurrency

```go
package main

import "fmt"

func worker(id int, jobs <-chan int, results chan<- int) {
    for j := range jobs {
        results <- j * 2
    }
}

func main() {
    jobs := make(chan int, 100)
    results := make(chan int, 100)

    for w := 1; w <= 3; w++ {
        go worker(w, jobs, results)
    }
}
```

## Shell Scripts

```bash
#!/bin/bash
set -euo pipefail

echo "Starting deployment..."
./deploy.sh --env production
```

## JavaScript

```javascript
const fetchUsers = async () => {
  const response = await fetch('/api/users');
  return response.json();
};
```

## Plain Code Block

```
This is a plain code block
with no language specified
```
