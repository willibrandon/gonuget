# HTTP Protocol Performance Benchmarks

This document presents performance benchmarks comparing HTTP/1.1, HTTP/2, and HTTP/3 protocols for the gonuget HTTP transport implementation.

## Test Environment

- **Platform**: darwin/arm64
- **CPU**: Apple M4 Pro
- **Go Version**: 1.21
- **Test Date**: 2025-10-23
- **Benchmark Duration**: 3 seconds per test

## Benchmark Results

### Sequential Requests (Single Request at a Time)

| Protocol | Latency (ns/op) | Memory (B/op) | Allocations (allocs/op) | vs HTTP/1.1 |
|----------|----------------|---------------|------------------------|-------------|
| HTTP/1.1 | 29,839         | 4,528         | 53                     | baseline    |
| HTTP/2   | 40,210         | 5,087         | 58                     | **+35%**    |
| HTTP/3   | 59,746         | 13,078        | 205                    | **+100%**   |

**Winner**: HTTP/1.1 (simplest protocol, lowest overhead)

### Concurrent Requests (Parallel with Connection Reuse)

| Protocol | Latency (ns/op) | Memory (B/op) | Allocations (allocs/op) | vs HTTP/1.1 | Speedup from Sequential |
|----------|----------------|---------------|------------------------|-------------|------------------------|
| HTTP/1.1 | 9,967          | 4,530         | 52                     | baseline    | **3.0x**               |
| HTTP/2   | 10,710         | 5,105         | 56                     | **+7%**     | **3.8x**               |
| HTTP/3   | 16,791         | 12,667        | 185                    | **+68%**    | **3.6x**               |

**Winner**: HTTP/1.1 (fastest raw performance, but requires many connections)

## Key Insights

### 1. Sequential Performance

HTTP/1.1 is the clear winner for single sequential requests:
- **29.8µs per request** - Simple protocol with minimal overhead
- HTTP/2 pays **35% penalty** for binary framing, HPACK compression, and flow control
- HTTP/3 pays **100% penalty** for QUIC protocol, QPACK compression, and per-packet encryption

### 2. Concurrent Performance

All protocols benefit significantly from parallelism, but in different ways:

- **HTTP/1.1**: 3.0x speedup via connection pooling
  - Requires many TCP connections (6-10 per host typically)
  - Risk of port exhaustion under high load
  - Each connection has separate TCP handshake overhead

- **HTTP/2**: 3.8x speedup via stream multiplexing
  - **Only 7% slower than HTTP/1.1** for concurrent requests
  - Uses **single TCP connection** with unlimited streams
  - No port exhaustion risk
  - Better resource utilization

- **HTTP/3**: 3.6x speedup via QUIC streams
  - 68% slower than HTTP/1.1 for concurrent requests
  - Uses **single QUIC connection** (UDP-based)
  - Provides unique benefits (see below)

### 3. Memory and Allocations

| Protocol | Memory Overhead | Allocation Overhead |
|----------|----------------|---------------------|
| HTTP/1.1 | baseline       | baseline            |
| HTTP/2   | +12%           | +8%                 |
| HTTP/3   | +180%          | +255%               |

HTTP/3 has significantly higher memory usage due to:
- QUIC connection state
- Packet loss recovery buffers
- Per-packet encryption overhead
- User-space protocol implementation

## Why HTTP/2 is the Production Winner

Despite being slower in raw benchmarks, HTTP/2 is superior for production use:

### Connection Efficiency

**HTTP/1.1 Concurrent**:
```
Client ----[TCP conn 1]----> Server
Client ----[TCP conn 2]----> Server
Client ----[TCP conn 3]----> Server
Client ----[TCP conn 4]----> Server
...
Client ----[TCP conn N]----> Server
```
- 6-10 connections per host (browser default)
- Each connection needs TCP handshake (1-3 RTTs)
- Risk of port exhaustion (65,535 limit)
- More memory per connection

**HTTP/2 Concurrent**:
```
Client =====[Single TCP conn]=====> Server
          [Stream 1, 2, 3, ..., N]
```
- **ONE connection** with unlimited streams
- Single TCP handshake
- No port exhaustion
- Efficient resource usage

### Real-World Benefits

1. **High-Latency Networks**
   - HTTP/1.1: Multiple handshakes = multiple RTTs
   - HTTP/2: Single handshake = saves RTTs

2. **CDN and Proxy Friendly**
   - HTTP/2's single connection reduces overhead on intermediaries
   - Better header compression with HPACK
   - Server push capability (optional)

3. **Head-of-Line Blocking**
   - HTTP/1.1: Sequential processing within each connection
   - HTTP/2: Parallel streams (but TCP-level HOL blocking remains)

4. **TLS Overhead**
   - HTTP/1.1: TLS handshake per connection
   - HTTP/2: Single TLS handshake

### Performance Trade-off Analysis

**7% performance cost for massive efficiency gains:**
```
HTTP/1.1: 100 requests = 10 connections × 10 requests/conn = fast but wasteful
HTTP/2:   100 requests = 1 connection × 100 streams      = 7% slower but efficient
```

The 7% overhead is **negligible** compared to:
- Network latency (typically 10-100ms)
- TLS handshake (10-50ms)
- DNS resolution (10-100ms)
- Server processing time (variable)

## Why HTTP/3 is Still Experimental

HTTP/3 is **68% slower** for concurrent requests but provides unique benefits:

### When HTTP/3 Wins

1. **Mobile Networks**
   - Connection migration: Handles IP address changes (WiFi → Cellular)
   - HTTP/1.1 and HTTP/2 drop connections on IP change

2. **High Packet Loss**
   - QUIC's forward error correction handles packet loss better than TCP
   - No TCP-level head-of-line blocking

3. **0-RTT Resumption**
   - Can resume connections with zero round trips
   - Faster than TCP Fast Open

4. **True Stream Independence**
   - Unlike HTTP/2, packet loss on one stream doesn't block others
   - UDP eliminates TCP's head-of-line blocking

### HTTP/3 Trade-offs

**Costs**:
- 68% slower latency
- 180% more memory
- 255% more allocations
- User-space protocol (not kernel-optimized)

**Benefits**:
- Connection migration
- Better packet loss handling
- No TCP head-of-line blocking
- Improved congestion control

## Recommendations

### Default Configuration

Use HTTP/2 as the default:

```go
config := http.DefaultTransportConfig()
// HTTP/2 is enabled by default
// config.EnableHTTP2 = true
// config.EnableHTTP3 = false
```

**Rationale**:
- Only 7% slower for concurrent requests
- Massive connection efficiency improvements
- Prevents port exhaustion
- Better for CDNs and proxies
- Mature and stable

### When to Enable HTTP/3

Enable HTTP/3 for specific use cases:

```go
config := http.TransportConfig{
    EnableHTTP2: false,
    EnableHTTP3: true,
}
```

**Use cases**:
- Mobile applications (connection migration)
- High packet loss environments
- Experimental/cutting-edge features
- When you specifically need QUIC benefits

### For NuGet Package Downloads

**Recommended**: HTTP/2 (default)

NuGet workloads typically involve:
- Sequential package downloads (one at a time)
- Large file transfers (packages can be MBs)
- Low-latency corporate/CI networks

HTTP/2 provides:
- Single connection efficiency
- Good enough performance (7% overhead acceptable)
- No port exhaustion issues
- Better compatibility with proxies/firewalls

## Running the Benchmarks

### All Benchmarks

```bash
go test ./http -bench=. -benchmem -benchtime=3s
```

### Sequential Only

```bash
go test ./http -bench='BenchmarkHTTP._Request$' -benchmem -benchtime=3s
```

### Concurrent Only

```bash
go test ./http -bench='ConcurrentRequests' -benchmem -benchtime=3s
```

### Compare Specific Protocols

```bash
# HTTP/1.1 vs HTTP/2
go test ./http -bench='BenchmarkHTTP[12]_' -benchmem

# All protocols sequential
go test ./http -bench='BenchmarkHTTP._Request$' -benchmem
```

## Benchmark Methodology

### Test Setup

**Sequential Benchmarks**:
- Single goroutine making requests serially
- Measures end-to-end request latency
- Connection reuse via http.Client

**Concurrent Benchmarks**:
- Uses `b.RunParallel()` to simulate concurrent load
- Connection pool warmed up with 10 requests
- `MaxIdleConnsPerHost: 100` for efficient reuse

### Test Servers

- **HTTP/1.1**: `httptest.NewServer` (plain HTTP)
- **HTTP/2**: `httptest.NewUnstartedServer` with `EnableHTTP2 = true`
- **HTTP/3**: Real `http3.Server` with UDP listener and TLS certificates

### What We Measure

1. **Latency (ns/op)**: Time per request including:
   - Protocol overhead (framing, compression)
   - Network stack (TCP/QUIC)
   - TLS encryption
   - Server processing

2. **Memory (B/op)**: Allocations per request including:
   - Request/response structures
   - Protocol buffers
   - Header compression tables

3. **Allocations (allocs/op)**: Number of heap allocations
   - Lower is better for GC pressure
   - HTTP/3 has 4x more allocations

## Conclusion

**The Winner: HTTP/2**

For gonuget's use case (NuGet package manager):
- ✅ HTTP/2 as default (only 7% slower, massively more efficient)
- ✅ HTTP/3 as opt-in (for mobile/experimental use)
- ❌ HTTP/1.1 only for compatibility (port exhaustion risk)

The benchmarks prove that raw performance isn't everything - connection efficiency, resource utilization, and real-world network behavior matter more for production systems.

---

**Last Updated**: 2025-10-23
**Test Platform**: Apple M4 Pro, darwin/arm64
