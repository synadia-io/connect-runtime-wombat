# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is the Wombat runtime for Synadia Connect - a runtime that wraps Wombat (a Benthos fork) to provide its extensive component ecosystem within the Connect platform. It bridges Connect's abstraction layer with Wombat/Benthos's rich set of data pipeline components.

Key characteristics:
- Provides 75 components (31 sources, 39 sinks, 5 scanners) from the Wombat/Benthos ecosystem
- Includes custom NATS-specific components not found in standard Benthos/RedPanda Connect
- Translates Connect specifications to Wombat/Benthos YAML configurations
- Integrates with Connect's metrics and execution model

## Build and Development Commands

### Building
```bash
# Build the runtime binary
task build  # Output: ./target/connect-runtime-wombat

# Build local Docker image
task docker:local  # Creates synadia/connect-runtime-wombat:local
```

### Testing
```bash
# Run all tests (excludes stress tests)
task test

# Run specific test categories
task test:nats        # NATS-specific component tests
task test:integration # Integration tests
task test:benchmark   # Performance benchmarks
task test:stress      # High-load stress tests
task test:coverage    # Generate coverage report

# Validate component specifications
task validate         # Validates all component YAML files against schema
```

### Linting
```bash
task lint  # Run golangci-lint
```

## Architecture and Code Structure

### Core Components

1. **Compiler** (`compiler/`): Translates Synadia Connect models to Wombat/Benthos YAML
   - Handles source/sink/transformer mappings
   - Converts Connect field specifications to Wombat format
   - Manages Bloblang script compilation

2. **Runner** (`runner/`): Executes Wombat configurations
   - Integrates metrics publishing to NATS
   - Manages runtime lifecycle
   - Handles configuration loading

3. **Components** (`components/`): Custom NATS-specific components
   - Additional sources/sinks not in standard Wombat
   - Enhanced NATS integration features

4. **Component Specifications** (`.connect/`):
   - `runtime.yml`: Runtime metadata and configuration
   - `sources/*.yml`: Source component specifications
   - `sinks/*.yml`: Sink component specifications
   - `scanners/*.yml`: Scanner component specifications

### Key Implementation Details

- **No Code Generation**: Unlike other Connect projects, components are hand-written YAML specs
- **Wombat Integration**: Leverages existing Wombat/Benthos functionality through configuration
- **Metrics**: Publishes to NATS subjects following NEX feed pattern (`synadia.connect.runtime.metrics`)
- **Component Discovery**: Crawls `.connect/` directories for available components

## Testing Philosophy

The project has comprehensive testing at multiple levels:

1. **Unit Tests**: Test individual functions and compilation logic
2. **Component Validation**: Ensures all 75 component specs are valid
3. **Integration Tests**: Test end-to-end flows with real NATS servers
4. **Benchmark Tests**: Measure throughput and latency performance
5. **Stress Tests**: Validate behavior under high load and concurrent operations

### Running Specific Tests
```bash
# Run tests for a specific package
ginkgo -v ./compiler

# Run a specific test file
ginkgo -v ./test/integration/sqs_test.go

# Run tests with specific labels
ginkgo -v -label-filter="integration" ./...
```

## Component Development

### Adding a New Component

1. Create component specification in appropriate directory:
   - Sources: `.connect/sources/my_component.yml`
   - Sinks: `.connect/sinks/my_component.yml`
   - Scanners: `.connect/scanners/my_component.yml`

2. Follow the component schema structure:
   ```yaml
   version: v1
   kind: ComponentSpec
   id: synadia:wombat:source:my_component
   title: My Component
   summary: Brief description
   description: |
     Detailed description
   config:
     fields:
       - name: field_name
         type: string
         description: Field description
         required: true
   ```

3. Validate the specification:
   ```bash
   task validate
   ```

4. Add tests in `test/component_test.go` to verify the component works correctly

### Wombat Configuration Translation

The compiler translates Connect configurations to Wombat format:
- Connect field names → Wombat field names
- Connect types → Wombat types
- Transformers → Bloblang processors
- Metrics integration → NATS metrics output

## Common Development Tasks

### Debugging Component Issues
```bash
# Enable verbose logging
CONNECT_LOG_LEVEL=debug ./target/connect-runtime-wombat config.json

# Check compiled Wombat configuration
# The compiler outputs the generated YAML which can be inspected
```

### Testing with Local NATS
```bash
# Start NATS server with JetStream
nats-server -js

# Or use the development server from connect-node
cd ../connect-node && task dev:server
```

### Performance Testing
```bash
# Run benchmarks for specific components
task test:benchmark

# Run stress tests to validate under load
task test:stress
```

## Troubleshooting

### Common Issues

1. **Component Not Found**: Ensure component YAML is in correct `.connect/` subdirectory
2. **Field Mapping Errors**: Check that Connect field names match Wombat expectations
3. **Bloblang Errors**: Validate transformer syntax using Wombat's Bloblang playground
4. **Metrics Not Publishing**: Verify NATS connection and subject permissions

### Debug Techniques

- Use `task validate` to check component specifications
- Enable debug logging to see compiled Wombat configurations
- Test components in isolation using Wombat's built-in test functionality
- Check NATS server logs for connection/permission issues
