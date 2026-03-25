# Load Testing for Isolate Panel

This directory contains load testing scripts using **k6**.

## Prerequisites

1. Install k6:
   ```bash
   # Ubuntu/Debian
   sudo gpg -k
   sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
   echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
   sudo apt-get update
   sudo apt-get install k6

   # macOS
   brew install k6

   # Windows (with Chocolatey)
   choco install k6
   ```

2. Start the Isolate Panel server in test mode:
   ```bash
   cd backend
   go run cmd/server/main.go
   ```

## Running Tests

### Run all tests
```bash
k6 run --config k6-config.json scenarios/*.js
```

### Run specific scenario
```bash
# API baseline test
k6 run scenarios/api-baseline.js

# Subscription export test
k6 run scenarios/subscription-export.js

# Concurrent writes test
k6 run scenarios/concurrent-writes.js

# Config generation test
k6 run scenarios/config-generation.js

# Stress test
k6 run scenarios/stress-test.js

# Endurance test (24 hours)
k6 run scenarios/endurance-test.js
```

### Run with custom configuration
```bash
k6 run --vus 50 --duration 10m scenarios/api-baseline.js
```

## Test Scenarios

| Scenario | Description | VUs | Duration | Target |
|----------|-------------|-----|----------|--------|
| **API Baseline** | Basic API endpoint performance | 100 | 30m | p95 < 100ms |
| **Subscription Export** | V2Ray/Clash/Singbox generation | 50 | 10m | p95 < 500ms |
| **Concurrent Writes** | Database write contention | 20 | 10m | No deadlocks |
| **Config Generation** | Core config generation | 10 | 10m | p95 < 200ms |
| **Stress Test** | Find breaking point | Ramp up | Until break | Document degradation |
| **Endurance Test** | Long-running stability | 100 | 24h | No memory leaks |

## Results

Test results are saved in `reports/` directory:
- `load-test-results.md` - Comprehensive results and analysis
- `*.json` - Raw k6 output data

## Interpreting Results

### Key Metrics

- **http_req_duration**: Response time percentiles
  - `p(95)`: 95th percentile (95% of requests faster than this)
  - `p(99)`: 99th percentile
  - `avg`: Average response time
  - `max`: Maximum response time

- **http_reqs**: Requests per second throughput

- **http_req_failed**: Failed request rate (should be < 1%)

- **vus**: Virtual users (concurrent connections)

### Acceptance Criteria

| Metric | Target | Warning | Critical |
|--------|--------|---------|----------|
| API p95 | < 100ms | 100-200ms | > 200ms |
| Subscription p95 | < 500ms | 500-1000ms | > 1000ms |
| Error rate | < 0.1% | 0.1-1% | > 1% |
| Memory usage | < 512MB | 512-768MB | > 768MB |

## Troubleshooting

### High Response Times

1. Check database query performance
2. Review connection pool settings
3. Analyze CPU and memory usage
4. Check for lock contention

### High Error Rates

1. Review server logs
2. Check database connection limits
3. Verify rate limiting configuration
4. Analyze error patterns

### Memory Leaks

1. Run endurance test (24h)
2. Monitor memory growth over time
3. Use `pprof` for heap analysis
4. Check for unclosed resources

## Integration with CI/CD

Load tests can be integrated into GitHub Actions:

```yaml
- name: Run load tests
  run: |
    k6 run --thresholds 'http_req_duration{expected_response:true}<200' scenarios/api-baseline.js
```

## Additional Resources

- [k6 Documentation](https://k6.io/docs/)
- [k6 JavaScript API](https://k6.io/docs/javascript-api/)
- [Performance Testing Best Practices](https://k6.io/blog/performance-testing-best-practices/)
