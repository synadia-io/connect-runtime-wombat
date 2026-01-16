package main

import (
	"regexp"
	"strings"
)

// toHumanReadableLabel converts a technical component name to a human-readable label
// Examples:
//   - amqp_0_9 → AMQP 0.9
//   - aws_s3 → AWS S3
//   - http_client → HTTP Client
func toHumanReadableLabel(name string) string {
	// Special cases that need custom handling
	specialCases := map[string]string{
		"amqp_0_9":             "AMQP 0.9",
		"amqp_1":               "AMQP 1",
		"aws_dynamodb":         "AWS DynamoDB",
		"aws_kinesis":          "AWS Kinesis",
		"aws_kinesis_firehose": "AWS Kinesis Firehose",
		"aws_s3":               "AWS S3",
		"aws_sns":              "AWS SNS",
		"aws_sqs":              "AWS SQS",
		"gcp_bigquery":         "GCP BigQuery",
		"gcp_cloud_storage":    "GCP Cloud Storage",
		"gcp_pubsub":           "GCP Pub/Sub",
		"http_client":          "HTTP Client",
		"http_server":          "HTTP Server",
		"nats_jetstream":       "NATS JetStream",
		"nats_kv":              "NATS KV",
		"redis_pubsub":         "Redis Pub/Sub",
		"redis_streams":        "Redis Streams",
		"redis_list":           "Redis List",
		"redis_scan":           "Redis Scan",
		"sql_insert":           "SQL Insert",
		"sql_select":           "SQL Select",
		"sql_raw":              "SQL Raw",
		"websocket":            "WebSocket",
		"stdin":                "STDIN",
		"stdout":               "STDOUT",
		"stderr":               "STDERR",
		"mqtt":                 "MQTT",
		"mongodb":              "MongoDB",
		"nanomsg":              "NanoMsg",
		"nats_stream":          "NATS Stream",
	}

	// Check if we have a special case
	if label, ok := specialCases[name]; ok {
		return label
	}

	// Known acronyms that should be uppercase
	acronyms := map[string]string{
		"amqp":   "AMQP",
		"aws":    "AWS",
		"gcp":    "GCP",
		"http":   "HTTP",
		"https":  "HTTPS",
		"sql":    "SQL",
		"nats":   "NATS",
		"mqtt":   "MQTT",
		"tcp":    "TCP",
		"udp":    "UDP",
		"tls":    "TLS",
		"ssl":    "SSL",
		"api":    "API",
		"csv":    "CSV",
		"json":   "JSON",
		"xml":    "XML",
		"sftp":   "SFTP",
		"ftp":    "FTP",
		"url":    "URL",
		"uri":    "URI",
		"uuid":   "UUID",
		"smtp":   "SMTP",
		"sns":    "SNS",
		"sqs":    "SQS",
		"s3":     "S3",
		"hdfs":   "HDFS",
		"nsq":    "NSQ",
		"nfs":    "NFS",
		"grpc":   "gRPC",
		"zeromq": "ZeroMQ",
		"zmq":    "ZMQ",
		"io":     "IO",
		"id":     "ID",
	}

	// Split by underscore
	parts := strings.Split(name, "_")

	// Process each part
	for i, part := range parts {
		// Check if it's a version number pattern (e.g., "0" followed by "9")
		if i > 0 && len(part) == 1 && regexp.MustCompile(`^\d$`).MatchString(part) &&
			i-1 < len(parts) && regexp.MustCompile(`^\d$`).MatchString(parts[i-1]) {
			// This is the second part of a version number, concatenate with dot
			parts[i-1] = parts[i-1] + "." + part
			parts[i] = "" // Mark for removal
			continue
		}

		// Check if it's a known acronym
		if upper, ok := acronyms[strings.ToLower(part)]; ok {
			parts[i] = upper
		} else {
			// Title case the word
			if len(part) > 0 {
				parts[i] = strings.ToUpper(part[:1]) + part[1:]
			}
		}
	}

	// Remove empty parts (from version number processing)
	var result []string
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}

	return strings.Join(result, " ")
}

// toHumanReadableFieldLabel converts a technical field name to a human-readable label
// This is simpler than component labels - just handle underscores and common acronyms
func toHumanReadableFieldLabel(name string) string {
	// Known acronyms that should be uppercase in field names
	fieldAcronyms := map[string]string{
		"url":    "URL",
		"urls":   "URLs",
		"uri":    "URI",
		"uris":   "URIs",
		"id":     "ID",
		"ids":    "IDs",
		"api":    "API",
		"apis":   "APIs",
		"http":   "HTTP",
		"https":  "HTTPS",
		"tls":    "TLS",
		"ssl":    "SSL",
		"tcp":    "TCP",
		"udp":    "UDP",
		"ip":     "IP",
		"dns":    "DNS",
		"ttl":    "TTL",
		"acl":    "ACL",
		"sql":    "SQL",
		"jwt":    "JWT",
		"oauth":  "OAuth",
		"oauth2": "OAuth2",
		"sasl":   "SASL",
		"ack":    "ACK",
		"nack":   "NACK",
		"qos":    "QoS",
		"dsn":    "DSN",
		"arn":    "ARN",
		"kms":    "KMS",
		"iam":    "IAM",
		"sqs":    "SQS",
		"sns":    "SNS",
		"s3":     "S3",
	}

	// First check if the entire name is a known acronym
	if upper, ok := fieldAcronyms[strings.ToLower(name)]; ok {
		return upper
	}

	// Split by underscore
	parts := strings.Split(name, "_")

	// Process each part
	for i, part := range parts {
		// Check if it's a known acronym
		if upper, ok := fieldAcronyms[strings.ToLower(part)]; ok {
			parts[i] = upper
		} else {
			// Title case the word
			if len(part) > 0 {
				parts[i] = strings.ToUpper(part[:1]) + part[1:]
			}
		}
	}

	return strings.Join(parts, " ")
}
