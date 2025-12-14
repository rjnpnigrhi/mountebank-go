# Feature Differences: Mountebank (JS) vs Mountebank-Go

This document outlines the feature parity and current gaps between the original [mountebank](http://www.mbtest.org/) (JavaScript/Node.js) and this Go port.

## ✅ Implemented Features

### Core
- **Architecture**: Multi-threaded, concurrent request handling using Go routines (vs Node.js single-threaded event loop).
- **API**: Compatible REST API for managing imposters and stubs.
- **CLI**: Basic `start`, `stop`, `restart` commands.
- **Docker**: Optimized Docker image (~20MB vs ~100MB+ for Node.js version).

### Protocols
- **HTTP**: Full support for HTTP/1.1.

### Predicates
- **equals**: Exact matching.
- **deepEquals**: Recursive object matching.
- **contains**: Substring/sub-object matching.
- **startsWith** / **endsWith**: String prefix/suffix matching.
- **matches**: Regex matching.
- **exists**: Field existence check.
- **not** / **or** / **and**: Logical operators.
- **Case Sensitivity**: Configurable case sensitivity.
- **Except**: Field exclusion regex.

### Behaviors
- **wait**: Latency injection.
- **copy**: Extracting values from the request and inserting them into the response.


### Advanced Matchers (Selectors)
- **JSONPath**: Selecting JSON nodes for matching.
- **XPath**: Selecting XML nodes for matching.

### JavaScript Injection
- **Predicate Injection**: Passing a JavaScript function to decide if a request matches.
- **Response Decoration**: Using `decorate` behavior to modify responses programmatically.
- **Middleware**: Global JavaScript middleware.

### Configuration & Persistence
- **Config Files**: Loading imposters from a config file (`--configfile`).
- **Save/Replay**: Saving current state to a file (`mb save`) and reloading it.

## 🚧 Missing / Planned Features

### Protocols
- **HTTPS**: TLS/SSL support, mutual authentication (mTLS).
- **TCP**: Raw TCP socket mocking (text and binary modes).
- **SMTP**: Email protocol mocking.
- **Proxying**: The ability to proxy requests to a real service and record responses (`proxyOnce`, `proxyAlways`, `proxyTransparent`).


### Advanced Matchers (Selectors)

### Advanced Behaviors

- **lookup**: Reading response data from external CSV files.
- **shellTransform**: Executing external shell scripts to generate responses.



### Miscellaneous
- **CORS**: Advanced CORS configuration is implemented for imposters.

## Performance Comparison

| Feature | JavaScript (Node.js) | Go |
|---------|---------------------|----|
| **Startup Time** | Slow (~1-2s) | Instant (<100ms) |
| **Memory Usage** | High (50MB+) | Low (<10MB) |
| **Concurrency** | Single-threaded (Event loop) | Multi-threaded (Goroutines) |
| **Binary Size** | N/A (Requires Node.js) | Single Binary (~15MB) |
| **Docker Image** | ~100MB+ | ~20MB |
