# NATS Component Testing Suite

This directory contains comprehensive tests for NATS components in the Wombat runtime, including integration tests, performance benchmarks, and stress tests.

## Test Structure

```
test/
├── integration/          # Functional integration tests
│   ├── nats_integration_test.go
│   └── integration_suite_test.go
├── benchmark/           # Performance benchmarks
│   └── nats_benchmark_test.go
├── stress/             # Stress and resilience tests
│   ├── nats_stress_test.go
│   └── stress_suite_test.go
└── run_nats_tests.sh   # Test runner script
```

## Running Tests

### Using Task (Recommended)

```bash
# Run all NATS tests (integration, benchmark, stress)
task test:nats

# Run only integration tests
task test:integration

# Run only benchmarks
task test:benchmark

# Run only stress tests
task test:stress

# Run all tests with coverage
task test:coverage
```

### Using the Test Script

```bash
# Run all NATS tests
./test/run_nats_tests.sh

# Run with coverage report
./test/run_nats_tests.sh --coverage
```

### Direct Execution

```bash
# Integration tests
ginkgo -v ./test/integration

# Benchmarks
go test -bench=. -benchmem ./test/benchmark -v

# Stress tests
ginkgo -v ./test/stress
```

## Test Coverage

### Integration Tests

The integration tests verify correct functionality of:

1. **Core NATS**
   - Publishing and subscribing to subjects
   - Queue groups
   - Message acknowledgment
   - JSON message handling

2. **JetStream**
   - Stream creation and management
   - Durable consumers
   - Message persistence
   - Acknowledgment handling

3. **JetStream KV**
   - Key-value storage operations
   - Watch functionality
   - Update notifications

### Performance Benchmarks

The benchmarks measure:

1. **Throughput Metrics**
   - Messages per second
   - MB/s data transfer rate

2. **Latency Metrics**
   - Average latency
   - P50, P95, P99 percentiles

3. **Test Scenarios**
   - Various message sizes (100B, 1KB, 10KB)
   - Sequential vs concurrent operations
   - Different concurrency levels (1, 10 publishers)

### Stress Tests

The stress tests validate system behavior under:

1. **High Volume**
   - 100K+ messages without loss
   - High throughput sustained operations

2. **Concurrent Operations**
   - 100+ concurrent publishers
   - Parallel consumers

3. **Large Messages**
   - 1MB+ message sizes
   - Memory efficiency

4. **Connection Resilience**
   - Automatic reconnection
   - Message delivery during network issues

5. **Resource Limits**
   - Maximum pending messages
   - Memory pressure
   - CPU saturation

## Benchmark Results Example

```
=== Small_1K_Sequential Benchmark Results ===
Messages: 1000
Duration: 245.123ms
Rate: 4081.23 msgs/sec
Throughput: 0.39 MB/s
Avg Latency: 1.2ms
P50 Latency: 1.1ms
P95 Latency: 2.3ms
P99 Latency: 3.1ms
================================
```

## Configuration

Tests use embedded NATS servers with:
- JetStream enabled
- Temporary storage directories
- Configurable limits for stress testing

## Requirements

- Go 1.21+
- Ginkgo v2 test framework
- NATS server dependencies (embedded)

## Debugging Failed Tests

1. **Integration Test Failures**
   ```bash
   # Run with verbose output
   ginkgo -v --fail-fast ./test/integration
   ```

2. **Benchmark Issues**
   ```bash
   # Run specific benchmark
   go test -bench=BenchmarkNATSCore/Small_1K_Sequential ./test/benchmark -v
   ```

3. **Stress Test Timeouts**
   - Check server resource limits
   - Verify JetStream storage configuration
   - Monitor memory usage during tests

## Adding New Tests

1. **New Integration Test**
   - Add test case to `nats_integration_test.go`
   - Follow existing patterns for setup/teardown
   - Use Ginkgo's `Describe` and `It` blocks

2. **New Benchmark**
   - Add benchmark function to `nats_benchmark_test.go`
   - Include various message sizes and concurrency levels
   - Report standard metrics (throughput, latency)

3. **New Stress Test**
   - Add scenario to `nats_stress_test.go`
   - Define clear failure criteria
   - Include recovery/resilience checks

## CI Integration

These tests are designed to run in CI environments:
- Embedded NATS server (no external dependencies)
- Configurable timeouts
- Clear pass/fail criteria
- Performance regression detection