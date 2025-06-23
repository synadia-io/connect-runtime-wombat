# Connect Runtime Wombat Examples

This document provides practical examples of using the connect-runtime-wombat to create data pipelines between various systems and NATS.

## Table of Contents

- [Basic Concepts](#basic-concepts)
- [Source/Sink Examples](#sourcesink-examples)
- [Producer/Consumer Examples](#producerconsumer-examples)
- [Transformer Examples](#transformer-examples)
- [Complete Pipeline Examples](#complete-pipeline-examples)

## Basic Concepts

The runtime compiles Connect specifications into Wombat configurations. There are two main patterns:

1. **Source → Sink**: Read from external system → Write to external system
2. **Consumer → Producer**: Read from NATS → Write to NATS

Both patterns support optional transformers for data processing.

## Source/Sink Examples

### HTTP API to S3

Read data from an HTTP endpoint and store it in S3:

```json
{
  "source": {
    "type": "http_client",
    "config": {
      "url": "https://api.example.com/data",
      "rate_limit": "1s",
      "headers": {
        "Authorization": "Bearer ${API_TOKEN}"
      }
    }
  },
  "sink": {
    "type": "s3",
    "config": {
      "bucket": "my-data-bucket",
      "path": "data/${!timestamp_unix()}.json",
      "region": "us-east-1"
    }
  }
}
```

### File to MongoDB

Process CSV files and insert into MongoDB:

```json
{
  "source": {
    "type": "file",
    "config": {
      "paths": ["/data/input/*.csv"],
      "codec": "csv"
    }
  },
  "transformer": {
    "mapping": {
      "script": """
      root.id = uuid_v4()
      root.timestamp = now()
      root.data = this
      """
    }
  },
  "sink": {
    "type": "mongodb",
    "config": {
      "url": "mongodb://localhost:27017",
      "database": "analytics",
      "collection": "events",
      "operation": "insert"
    }
  }
}
```

## Producer/Consumer Examples

### Core NATS Pub/Sub

Simple message relay between subjects:

```json
{
  "consumer": {
    "core": {
      "subject": "orders.new",
      "queue": "order-processors"
    },
    "nats": {
      "url": "nats://localhost:4222"
    }
  },
  "producer": {
    "core": {
      "subject": "orders.processed"
    },
    "nats": {
      "url": "nats://localhost:4222"
    }
  }
}
```

### JetStream Consumer/Producer

Process messages from a JetStream with exactly-once semantics:

```json
{
  "consumer": {
    "stream": {
      "stream": "ORDERS",
      "consumer": "order-processor",
      "ack_wait": "30s",
      "max_deliver": 3
    },
    "nats": {
      "url": "nats://localhost:4222"
    }
  },
  "transformer": {
    "mapping": {
      "script": """
      root = this
      root.processed_at = now()
      root.processor_id = env("PROCESSOR_ID")
      """
    }
  },
  "producer": {
    "stream": {
      "stream": "PROCESSED_ORDERS",
      "subject": "processed.orders"
    },
    "nats": {
      "url": "nats://localhost:4222"
    }
  }
}
```

### Key-Value Store Operations

Read from one KV store and write to another:

```json
{
  "consumer": {
    "kv": {
      "bucket": "config",
      "key": "settings.*",
      "watch": true
    },
    "nats": {
      "url": "nats://localhost:4222"
    }
  },
  "producer": {
    "kv": {
      "bucket": "replicated-config",
      "key": "${! meta("nats_kv_key") }"
    },
    "nats": {
      "url": "nats://backup-cluster:4222"
    }
  }
}
```

## Transformer Examples

### Service Transformer

Call an external NATS service for enrichment:

```json
{
  "transformer": {
    "service": {
      "endpoint": "enrichment.user",
      "timeout": "5s",
      "nats": {
        "url": "nats://localhost:4222"
      }
    }
  }
}
```

### Composite Transformer

Chain multiple transformations:

```json
{
  "transformer": {
    "composite": {
      "sequential": [
        {
          "mapping": {
            "script": "root = this.filter(v -> v.status == \"active\")"
          }
        },
        {
          "service": {
            "endpoint": "validation.service",
            "timeout": "2s",
            "nats": {
              "url": "nats://localhost:4222"
            }
          }
        },
        {
          "mapping": {
            "script": """
            root.id = uuid_v4()
            root.timestamp = now()
            root.original = this
            """
          }
        }
      ]
    }
  }
}
```

### Explode Transformer

Split arrays into individual messages:

```json
{
  "transformer": {
    "explode": {
      "type": "root",
      "path": "items"
    }
  }
}
```

### Combine Transformer

Batch messages together:

```json
{
  "transformer": {
    "combine": {
      "type": "archive",
      "count": 100,
      "period": "10s"
    }
  }
}
```

## Complete Pipeline Examples

### Real-time Analytics Pipeline

Collect metrics from HTTP endpoints, enrich them, and store in both NATS and S3:

```json
{
  "source": {
    "type": "http_client",
    "config": {
      "url": "https://metrics.example.com/api/v1/metrics",
      "rate_limit": "5s",
      "retry_after": "1s",
      "max_retry_backoff": "30s"
    }
  },
  "transformer": {
    "composite": {
      "sequential": [
        {
          "mapping": {
            "script": """
            let metrics = this.metrics.map(m -> {
              "name": m.name,
              "value": m.value,
              "tags": m.tags,
              "timestamp": now(),
              "source": "api"
            })
            root = {"metrics": metrics}
            """
          }
        },
        {
          "explode": {
            "type": "root",
            "path": "metrics"
          }
        },
        {
          "service": {
            "endpoint": "enrichment.metrics",
            "timeout": "2s",
            "nats": {
              "url": "nats://localhost:4222"
            }
          }
        }
      ]
    }
  },
  "sink": {
    "type": "broker",
    "config": {
      "outputs": [
        {
          "nats": {
            "urls": ["nats://localhost:4222"],
            "subject": "metrics.enriched"
          }
        },
        {
          "s3": {
            "bucket": "metrics-archive",
            "path": "year=${!timestamp().format("2006")}/month=${!timestamp().format("01")}/day=${!timestamp().format("02")}/metrics-${!timestamp_unix()}.json",
            "region": "us-east-1",
            "batching": {
              "count": 1000,
              "period": "1m"
            }
          }
        }
      ]
    }
  }
}
```

### Change Data Capture (CDC) Pipeline

Stream database changes to NATS:

```json
{
  "source": {
    "type": "kafka",
    "config": {
      "addresses": ["kafka1:9092", "kafka2:9092"],
      "topics": ["mysql.inventory.customers"],
      "consumer_group": "cdc-to-nats",
      "client_id": "connect-runtime"
    }
  },
  "transformer": {
    "mapping": {
      "script": """
      let op = match this.payload.op {
        "c" => "INSERT",
        "u" => "UPDATE",
        "d" => "DELETE",
        _ => "UNKNOWN"
      }

      root = {
        "operation": op,
        "table": this.payload.source.table,
        "timestamp": this.payload.ts_ms,
        "before": this.payload.before,
        "after": this.payload.after,
        "key": this.payload.key
      }
      """
    }
  },
  "producer": {
    "stream": {
      "stream": "CDC_EVENTS",
      "subject": "cdc.${!json("table")}",
      "threads": 10
    },
    "nats": {
      "url": "nats://localhost:4222"
    }
  }
}
```

### IoT Data Collection

Collect MQTT sensor data and store in time-series format:

```json
{
  "source": {
    "type": "mqtt",
    "config": {
      "urls": ["tcp://mqtt.broker:1883"],
      "topics": ["sensors/+/temperature", "sensors/+/humidity"],
      "client_id": "connect-iot-collector",
      "qos": 1
    }
  },
  "transformer": {
    "mapping": {
      "script": """
      let parts = meta("mqtt_topic").split("/")
      root = {
        "sensor_id": parts.index(1),
        "metric_type": parts.index(2),
        "value": this.value.number(),
        "timestamp": this.timestamp,
        "location": this.location,
        "metadata": {
          "unit": this.unit,
          "quality": meta("mqtt_qos")
        }
      }
      """
    }
  },
  "producer": {
    "stream": {
      "stream": "SENSOR_DATA",
      "subject": "sensors.${!json("sensor_id")}.${!json("metric_type")}"
    },
    "nats": {
      "url": "nats://localhost:4222"
    }
  }
}
```

## Configuration Best Practices

1. **Error Handling**: Always configure retry policies and dead letter queues
2. **Resource Management**: Set appropriate rate limits and batch sizes
3. **Monitoring**: Use the built-in metrics publishing to track pipeline health
4. **Security**: Use JWT/Seed authentication for NATS connections
5. **Scalability**: Configure thread counts based on workload

## Environment Variables

Common environment variables used in configurations:

- `NATS_URL`: NATS server URL
- `NATS_JWT`: Authentication JWT
- `NATS_SEED`: Authentication seed
- `PROCESSOR_ID`: Unique identifier for the processor instance
- `API_TOKEN`: External API authentication token

## Troubleshooting

Enable debug logging by setting:
```bash
export CONNECT_LOG_LEVEL=debug
```

View the compiled Wombat configuration in the logs (base64 encoded) to debug compilation issues.
