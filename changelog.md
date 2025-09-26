# Changelog

Definitions: 
* `added`: Introduced an option to a component
* `ignored`: Not exposing an option present in the wrapped component
* `deprecated`: Option is still present on component, but ignored by wrapped component
* `missing`: Full component is missing from Connect
* `new`: Newly introduced component

## outputs
### amqp_0_9
* added `exchange_declare.arguments` ([docs](https://wombat.dev/reference/components/outputs/amqp_0_9/#exchange_declarearguments))

### aws_s3, aws_sns, aws_sqs, gcp_pubsub
* changed type of `metadata.metadata.exclude_prefixes` from `string` to `list(string)` ([source](https://github.com/redpanda-data/benthos/blob/70b2d1fe4fa5eb2e0de38d1d42460f70a7bae5ec/internal/metadata/exclude_filter.go#L14))

### sql_raw
* new

### snowflake_put
* new

## inputs

### amqp_0_9
* added `queue_declare.arguments` ([docs](https://wombat.dev/reference/components/inputs/amqp_0_9/#queue_declarearguments))

### gcp_bigquery_select
* added `prefix` ([docs](https://wombat.dev/reference/components/inputs/gcp_bigquery_select/#prefix))
* added `suffix`

### sql_raw
* new
---

# non-exposed options 

## outputs 

### azure_cosmosdb
* ignored `endpoint` ([docs](https://wombat.dev/reference/components/outputs/azure_cosmosdb/#endpoint))
* ignored `account_key`

### aws_dynamodb
* ignored aws credential options `credentials.profile`, `credentials.role`, `credentials.from_ec2_role`, `credentials.role_external_id` [docs](https://wombat.dev/reference/components/outputs/aws_dynamodb/#credentials-1)
* ignored `batching` config [docs](https://docs.redpanda.com/redpanda-connect/configuration/batching/)
* ignored `backoff` config [docs](https://wombat.dev/reference/components/outputs/aws_dynamodb/#backoff)

### cassandra
* ignored `batching.processors` [docs](https://wombat.dev/reference/components/outputs/cassandra/#batchingprocessors)
* ignored `host_selection_policy` [docs](https://wombat.dev/reference/components/outputs/cassandra/#host_selection_policy)

### couchbase
* ignored `batching.processors`
* added `scope` [docs](https://wombat.dev/reference/components/outputs/couchbase/#scope)

### cypher
* ignored `batching.processors`
* deprecated `tls.tls.enabled` [source](https://github.com/redpanda-data/benthos/blob/v4.56.0/public/service/config_tls.go#L24)

### gcp_bigquery
* ignored `csv` [docs](https://wombat.dev/reference/components/outputs/gcp_bigquery/#csv-1)


### gcp_pubsub
* ignored flow_control [docs](https://wombat.dev/reference/components/outputs/gcp_pubsub/#flow_control)

### hdfs
* ignored `batching.processors`

### http_client
* ignored `batching.processors`
* ignored `jwt`
* ignored `rate_limit`

### kafka_franz
* ignored `batching.processors`


### mongodb
* ignored `write_concern` [docs](https://wombat.dev/reference/components/outputs/mongodb/#write_concern)

### nats
* ignored `auth`
* ignored `max_reconnects`
* ignored `tls`
* ignored `inject_tracing_map`

### nats_jetstream
* ignored `auth`
* ignored `max_reconnects`
* ignored `tls`
* ignored `inject_tracing_map`

### nats_kv
* ignored `auth`
* ignored `max_reconnects`
* ignored `tls`
* ignored `inject_tracing_map`

### opensearch
* ignored `batching.processors`

### pulsar
* ignored `tls`

### pusher
* ignored `batching.processors`

### redis_list
* ignored `batching.processors`

### redis_pubsub
* ignored `batching.processors`

### redis_streams
* ignored `batching.processors`

### sftp
* ignored `credentials`

### splunk_hec
* missing

### sql 
* missing 

### sql_insert
* missing 

### sql_raw
* ignored `batching.processors`

### snowflake_put
* ignored `batching.processors`
* ignored `private_key_file`
* ignored `private_key_pass`

## Inputs

### azure_cosmosdb
* ignored `endpoint` ([docs](https://wombat.dev/reference/components/inputs/azure_cosmosdb/#endpoint))
* ignored `account_key`

### cockroachdb_changefeed
* missing

### discord
* missing

### aws_kinesis
* ignored `dynamodb.region`
* ignored `dynamodb.endpoint`
* ignored `dynamodb.credentials`

### azure_blob_storage
* ignored `targets_input`

### cassandra
* ignored `host_selection_policy` [docs](https://wombat.dev/reference/components/inputs/cassandra/#host_selection_policy)

### gcp_pubsub
* ignored `create_subscription` [docs](https://wombat.dev/reference/components/inputs/gcp_pubsub/#create_subscription)

### http_client
* ignored `jwt`
* ignored `rate_limit`

### kafka_franz
* ignored `batching.processors`
* deprecated `start_from_oldest` (replaced by `start_offset`) ([docs](https://wombat.dev/reference/components/inputs/kafka_franz/#start_offset))

### mqtt
* ignored `tls` 

### nats
* ignored `auth`
* ignored `max_reconnects`
* ignored `tls`
* ignored `extract_tracing_map`

### nats_jetstream
* ignored `auth`
* ignored `max_reconnects`
* ignored `tls`
* ignored `extract_tracing_map`

### nats_kv
* ignored `auth`
* ignored `max_reconnects`
* ignored `tls`
* ignored `extract_tracing_map`

### pulsar
* ignored `tls`

### sftp
* missing

### socket
* ignored `tls`
* ignored `open_message_mapping`

### spicedb_watch
* missing

### sql_select
* missing