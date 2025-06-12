package benchmark_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/redpanda-data/benthos/v4/public/service"

	// Import all components
	_ "github.com/synadia-io/connect-runtime-wombat/components"
)

// Simple benchmarks that run quickly and don't overwhelm the system

func init() {
	// Benchmarks will run with default logging
	// To disable logging, set BENTHOS_LOG_LEVEL=none environment variable
}

func BenchmarkSimpleNATSCore(b *testing.B) {
	// Setup NATS server once
	opts := test.DefaultTestOptions
	opts.Port = -1
	srv := test.RunServer(&opts)
	defer srv.Shutdown()

	b.Run("Publish_100_Messages", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchmarkNATSCorePublish(srv.ClientURL(), 100)
		}
	})

	b.Run("Publish_1000_Messages", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchmarkNATSCorePublish(srv.ClientURL(), 1000)
		}
	})
}

func benchmarkNATSCorePublish(natsURL string, messageCount int) {
	subject := "bench.simple"

	// Producer configuration
	producerConfig := fmt.Sprintf(`
input:
  generate:
    count: %d
    interval: "0s"
    mapping: |
      root.id = counter()
      root.data = "test_message_" + counter().string()

output:
  nats:
    urls: ["%s"]
    subject: "%s"
`, messageCount, natsURL, subject)

	// Create and run producer
	builder := service.NewStreamBuilder()
	if err := builder.SetYAML(producerConfig); err != nil {
		panic(err)
	}

	stream, err := builder.Build()
	if err != nil {
		panic(err)
	}

	if err := stream.Run(context.Background()); err != nil {
		panic(err)
	}
}

func BenchmarkSimpleJetStream(b *testing.B) {
	// Setup NATS server with JetStream
	opts := test.DefaultTestOptions
	opts.Port = -1
	opts.JetStream = true
	opts.StoreDir = b.TempDir()
	srv := test.RunServer(&opts)
	defer srv.Shutdown()

	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		b.Fatal(err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		b.Fatal(err)
	}

	// Create stream once
	_, err = js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     "BENCH_STREAM",
		Subjects: []string{"bench.js.>"},
	})
	if err != nil {
		b.Fatal(err)
	}

	b.Run("Publish_100_Messages", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchmarkJetStreamPublish(srv.ClientURL(), 100)
		}
	})

	b.Run("Publish_1000_Messages", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchmarkJetStreamPublish(srv.ClientURL(), 1000)
		}
	})
}

func benchmarkJetStreamPublish(natsURL string, messageCount int) {
	subject := "bench.js.test"

	// Producer configuration
	producerConfig := fmt.Sprintf(`
input:
  generate:
    count: %d
    interval: "0s"
    mapping: |
      root.id = counter()
      root.data = "jetstream_message_" + counter().string()

output:
  nats_jetstream:
    urls: ["%s"]
    subject: "%s"
`, messageCount, natsURL, subject)

	// Create and run producer
	builder := service.NewStreamBuilder()
	if err := builder.SetYAML(producerConfig); err != nil {
		panic(err)
	}

	stream, err := builder.Build()
	if err != nil {
		panic(err)
	}

	if err := stream.Run(context.Background()); err != nil {
		panic(err)
	}
}

func BenchmarkSimpleKV(b *testing.B) {
	// Setup NATS server with JetStream
	opts := test.DefaultTestOptions
	opts.Port = -1
	opts.JetStream = true
	opts.StoreDir = b.TempDir()
	srv := test.RunServer(&opts)
	defer srv.Shutdown()

	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		b.Fatal(err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		b.Fatal(err)
	}

	// Create KV bucket once
	_, err = js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
		Bucket: "BENCH_KV",
	})
	if err != nil {
		b.Fatal(err)
	}

	b.Run("Write_10_Keys", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchmarkKVWrite(srv.ClientURL(), 10)
		}
	})

	b.Run("Write_100_Keys", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchmarkKVWrite(srv.ClientURL(), 100)
		}
	})
}

func benchmarkKVWrite(natsURL string, keyCount int) {
	// Producer configuration
	producerConfig := fmt.Sprintf(`
input:
  generate:
    count: %d
    interval: "0s"
    mapping: |
      root = {
        "key": "bench_key_" + counter().string(),
        "value": "test_value_" + counter().string()
      }

output:
  nats_kv:
    urls: ["%s"]
    bucket: "BENCH_KV"
    key: "${! json(\"key\") }"
`, keyCount, natsURL)

	// Create and run producer
	builder := service.NewStreamBuilder()
	if err := builder.SetYAML(producerConfig); err != nil {
		panic(err)
	}

	stream, err := builder.Build()
	if err != nil {
		panic(err)
	}

	if err := stream.Run(context.Background()); err != nil {
		panic(err)
	}
}

// Throughput test
func TestNATSThroughput(t *testing.T) {
	// Setup NATS server
	opts := test.DefaultTestOptions
	opts.Port = -1
	srv := test.RunServer(&opts)
	defer srv.Shutdown()

	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()

	messageCount := 10000
	subject := "throughput.test"
	var received atomic.Int32

	// Subscribe
	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		received.Add(1)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = sub.Unsubscribe()
	}()

	// Producer configuration
	producerConfig := fmt.Sprintf(`
input:
  generate:
    count: %d
    interval: "0s"
    mapping: |
      root = range(0, 1024).map_each(_ -> "x").join("")  # 1KB message

output:
  nats:
    urls: ["%s"]
    subject: "%s"
    max_in_flight: 100
`, messageCount, srv.ClientURL(), subject)

	// Create and run producer
	builder := service.NewStreamBuilder()
	if err := builder.SetYAML(producerConfig); err != nil {
		t.Fatal(err)
	}

	stream, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	if err := stream.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	// Wait for messages
	time.Sleep(1 * time.Second)
	duration := time.Since(start)

	t.Logf("Published %d messages in %v", messageCount, duration)
	t.Logf("Rate: %.2f msgs/sec", float64(messageCount)/duration.Seconds())
	t.Logf("Throughput: %.2f MB/s", float64(messageCount*1024)/(1024*1024)/duration.Seconds())
	t.Logf("Received: %d messages", received.Load())
}
