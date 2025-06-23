# Connect Runtime Wombat API Reference

## Overview

The connect-runtime-wombat provides a runtime environment for executing Synadia Connect specifications using the Wombat/Benthos engine. This document details the API and configuration options.

## Table of Contents

- [Runtime Configuration](#runtime-configuration)
- [Specification Format](#specification-format)
- [Source/Sink Configuration](#sourcesink-configuration)
- [Producer/Consumer Configuration](#producerconsumer-configuration)
- [Transformer Configuration](#transformer-configuration)
- [Metrics Configuration](#metrics-configuration)
- [Component Reference](#component-reference)

## Runtime Configuration

The runtime is configured through environment variables set by the Connect platform:

| Variable | Description | Required |
|----------|-------------|----------|
| `NATS_URL` | NATS server URL | Yes |
| `NATS_JWT` | Authentication JWT | No |
| `NATS_SEED` | Authentication seed | No |
| `NAMESPACE` | Account/namespace name | Yes |
| `CONNECTOR_ID` | Unique connector identifier | Yes |
| `INSTANCE_ID` | Instance identifier | Yes |

## Specification Format

Connect specifications are JSON documents with the following structure:

```typescript
interface ConnectorSpec {
  // Either source/sink OR consumer/producer (not both)
  source?: SourceStep;
  sink?: SinkStep;
  consumer?: ConsumerStep;
  producer?: ProducerStep;

  // Optional transformer
  transformer?: TransformerStep;
}
```

## Source/Sink Configuration

### SourceStep

Reads data from external systems:

```typescript
interface SourceStep {
  type: string;           // Component type (e.g., "http_client", "file", "s3")
  config: object;         // Component-specific configuration
}
```

### SinkStep

Writes data to external systems:

```typescript
interface SinkStep {
  type: string;           // Component type (e.g., "http_server", "file", "s3")
  config: object;         // Component-specific configuration
}
```

## Producer/Consumer Configuration

### ConsumerStep

Reads messages from NATS:

```typescript
interface ConsumerStep {
  nats: NatsConfig;       // NATS connection configuration

  // Exactly one of these must be specified:
  core?: CoreConsumer;    // Core NATS consumer
  stream?: StreamConsumer; // JetStream consumer
  kv?: KvConsumer;        // Key-Value consumer
}
```

#### CoreConsumer

```typescript
interface CoreConsumer {
  subject: string;        // NATS subject to subscribe to
  queue?: string;         // Optional queue group name
}
```

#### StreamConsumer

```typescript
interface StreamConsumer {
  stream: string;         // JetStream stream name
  consumer: string;       // Consumer name
  ack_wait?: string;      // Acknowledgment wait time (default: "30s")
  max_deliver?: number;   // Maximum delivery attempts
  filter_subject?: string; // Subject filter
}
```

#### KvConsumer

```typescript
interface KvConsumer {
  bucket: string;         // KV bucket name
  key: string;            // Key pattern to watch (supports wildcards)
  watch?: boolean;        // Watch for changes (default: true)
}
```

### ProducerStep

Writes messages to NATS:

```typescript
interface ProducerStep {
  nats: NatsConfig;       // NATS connection configuration
  threads: number;        // Parallelism level (default: 1)

  // Exactly one of these must be specified:
  core?: CoreProducer;    // Core NATS producer
  stream?: StreamProducer; // JetStream producer
  kv?: KvProducer;        // Key-Value producer
}
```

#### CoreProducer

```typescript
interface CoreProducer {
  subject: string;        // NATS subject to publish to
}
```

#### StreamProducer

```typescript
interface StreamProducer {
  stream: string;         // JetStream stream name
  subject: string;        // Subject to publish to
}
```

#### KvProducer

```typescript
interface KvProducer {
  bucket: string;         // KV bucket name
  key: string;            // Key to set (supports interpolation)
}
```

### NatsConfig

Common NATS configuration:

```typescript
interface NatsConfig {
  url?: string;           // NATS server URL (overrides runtime default)
  jwt?: string;           // Authentication JWT (overrides runtime default)
  seed?: string;          // Authentication seed (overrides runtime default)
}
```

## Transformer Configuration

Transformers modify messages as they flow through the pipeline:

```typescript
interface TransformerStep {
  // Exactly one of these must be specified:
  mapping?: MappingTransformer;
  service?: ServiceTransformer;
  composite?: CompositeTransformer;
  explode?: ExplodeTransformer;
  combine?: CombineTransformer;
}
```

### MappingTransformer

Transform messages using Bloblang scripts:

```typescript
interface MappingTransformer {
  script: string;         // Bloblang transformation script
}
```

### ServiceTransformer

Call external NATS service:

```typescript
interface ServiceTransformer {
  endpoint: string;       // NATS subject for request/reply
  timeout: string;        // Request timeout (e.g., "5s")
  nats: NatsConfig;       // NATS connection configuration
}
```

### CompositeTransformer

Chain multiple transformers:

```typescript
interface CompositeTransformer {
  sequential: TransformerStep[]; // Transformers to apply in order
}
```

### ExplodeTransformer

Split arrays/objects into individual messages:

```typescript
interface ExplodeTransformer {
  type: "root" | "json_array" | "json_object";
  path?: string;          // JSONPath to array/object (for type "root")
}
```

### CombineTransformer

Batch messages together:

```typescript
interface CombineTransformer {
  type: "archive" | "archive_zip" | "json_array";
  count?: number;         // Max messages per batch
  period?: string;        // Max time to wait (e.g., "10s")
}
```

## Metrics Configuration

Metrics are automatically published to NATS if runtime configuration is provided:

- **Subject**: `$NEX.FEED.<namespace>.metrics.<instance_id>`
- **Format**: Prometheus text format
- **Interval**: 5 seconds
- **Headers**:
  - `account`: Namespace/account name
  - `connector_id`: Connector identifier
  - `instance_id`: Instance identifier

## Component Reference

### Available Components

The runtime includes all standard Wombat/Benthos components plus custom NATS components:

#### Sources (31 total)
- Cloud: `aws_s3`, `aws_sqs`, `aws_kinesis`, `azure_blob_storage`, `gcp_pubsub`
- Databases: `mongodb`, `sql_select`, `cassandra`, `redis_streams`
- Messaging: `kafka`, `amqp_0_9`, `mqtt`, `nsq`, `pulsar`
- Files: `file`, `sftp`, `hdfs`
- Network: `http_client`, `http_server`, `websocket`
- NATS: `nats` (custom enhanced component)

#### Sinks (39 total)
- Cloud: `aws_s3`, `aws_sqs`, `aws_sns`, `aws_kinesis`, `azure_blob_storage`
- Databases: `mongodb`, `sql_insert`, `cassandra`, `redis_*`, `elasticsearch`
- Messaging: `kafka`, `amqp_0_9`, `mqtt`, `nsq`, `pulsar`
- Files: `file`, `sftp`, `hdfs`
- Network: `http_client`, `websocket`, `socket`
- NATS: `nats` (custom enhanced component)

#### Scanners (5 total)
- `aws_s3`
- `azure_blob_storage`
- `file`
- `gcp_cloud_storage`
- `sftp`

### Bloblang Functions

All standard Bloblang functions are available for use in mapping transformers:

- String manipulation: `uppercase()`, `lowercase()`, `trim()`, `split()`, etc.
- JSON operations: `json()`, `parse_json()`, `format_json()`
- Time functions: `now()`, `timestamp_unix()`, `parse_timestamp()`
- Cryptographic: `hash()`, `hmac()`, `uuid_v4()`
- Encoding: `base64_encode()`, `base64_decode()`, `url_encode()`
- Metadata: `meta()`, `root_meta()`, `env()`

### Interpolation

String fields support interpolation using `${!expression}` syntax:

```json
{
  "subject": "events.${!json(\"type\")}.${!timestamp().format(\"2006-01-02\")}"
}
```

## Error Handling

- All components support configurable retry policies
- Failed messages can be routed to dead letter queues
- Detailed error information is available in message metadata

## Performance Tuning

Key configuration options for performance:

1. **Thread Count**: Set `threads` on producers for parallelism
2. **Batching**: Configure batch sizes and timeouts for sinks
3. **Rate Limiting**: Use `rate_limit` on sources to control throughput
4. **Buffer Sizes**: Configure internal buffer sizes for components
5. **Connection Pooling**: Most components support connection pooling

## Security

- Use JWT/Seed authentication for NATS connections
- TLS is supported for all network components
- Credentials can be passed via environment variables
- IAM roles supported for AWS components
