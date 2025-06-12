package benchmark_test

import (
	"context"
	"fmt"
	"sync"
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

// BenchmarkResults holds performance metrics
type BenchmarkResults struct {
	MessageCount     int
	Duration         time.Duration
	MessagesPerSec   float64
	AvgLatency       time.Duration
	P50Latency       time.Duration
	P95Latency       time.Duration
	P99Latency       time.Duration
	BytesTransferred int64
	Throughput       float64 // MB/s
}

func (r *BenchmarkResults) Print(name string) {
	fmt.Printf("\n=== %s Benchmark Results ===\n", name)
	fmt.Printf("Messages: %d\n", r.MessageCount)
	fmt.Printf("Duration: %v\n", r.Duration)
	fmt.Printf("Rate: %.2f msgs/sec\n", r.MessagesPerSec)
	fmt.Printf("Throughput: %.2f MB/s\n", r.Throughput)
	fmt.Printf("Avg Latency: %v\n", r.AvgLatency)
	fmt.Printf("P50 Latency: %v\n", r.P50Latency)
	fmt.Printf("P95 Latency: %v\n", r.P95Latency)
	fmt.Printf("P99 Latency: %v\n", r.P99Latency)
	fmt.Printf("================================\n")
}

func BenchmarkNATSCore(b *testing.B) {
	benchmarks := []struct {
		name         string
		messageCount int
		messageSize  int
		concurrent   int
	}{
		{"Small_1K_Sequential", 1000, 100, 1},
		{"Small_10K_Sequential", 10000, 100, 1},
		{"Small_10K_Concurrent", 10000, 100, 10},
		{"Medium_1K_Sequential", 1000, 1024, 1},
		{"Medium_10K_Concurrent", 10000, 1024, 10},
		{"Large_1K_Sequential", 1000, 10240, 1},
		{"Large_1K_Concurrent", 1000, 10240, 10},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			// Setup NATS server
			opts := test.DefaultTestOptions
			opts.Port = -1
			srv := test.RunServer(&opts)
			defer srv.Shutdown()

			nc, err := nats.Connect(srv.ClientURL())
			if err != nil {
				b.Fatal(err)
			}
			defer nc.Close()

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				results := benchmarkNATSCore(srv.ClientURL(), bm.messageCount, bm.messageSize, bm.concurrent)
				_ = results
			}
		})
	}
}

func benchmarkNATSCore(natsURL string, messageCount, messageSize, concurrent int) *BenchmarkResults {
	subject := "bench.core"
	received := make(chan time.Duration, messageCount)
	var receivedCount atomic.Int32
	var bytesReceived atomic.Int64

	nc, err := nats.Connect(natsURL)
	if err != nil {
		panic(err)
	}
	defer nc.Close()

	// Subscribe to measure latency
	startTime := time.Now()
	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		// Extract timestamp from message
		sentTime := time.Unix(0, int64(msg.Header.Get("timestamp")[0]))
		latency := time.Since(sentTime)
		received <- latency
		receivedCount.Add(1)
		bytesReceived.Add(int64(len(msg.Data)))
	})
	if err != nil {
		panic(err)
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
      root.data = "test_payload_" + counter().string()
      root.size = %d

output:
  nats:
    urls: ["%s"]
    subject: "%s"
    headers:
      timestamp: "${! (timestamp_unix_nano()).string() }"
    max_in_flight: %d
`, messageCount, messageSize, natsURL, subject, concurrent)

	// Create and run producer
	builder := service.NewStreamBuilder()
	if err := builder.SetYAML(producerConfig); err != nil {
		panic(err)
	}

	stream, err := builder.Build()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	if err := stream.Run(ctx); err != nil {
		panic(err)
	}

	// Wait for all messages
	timeout := time.After(30 * time.Second)
	for receivedCount.Load() < int32(messageCount) {
		select {
		case <-timeout:
			panic(fmt.Sprintf("timeout: received only %d/%d messages", receivedCount.Load(), messageCount))
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	duration := time.Since(startTime)
	close(received)

	// Calculate latencies
	latencies := make([]time.Duration, 0, messageCount)
	var totalLatency time.Duration
	for lat := range received {
		latencies = append(latencies, lat)
		totalLatency += lat
	}

	// Sort latencies for percentile calculation
	sortDurations(latencies)

	results := &BenchmarkResults{
		MessageCount:     messageCount,
		Duration:         duration,
		MessagesPerSec:   float64(messageCount) / duration.Seconds(),
		AvgLatency:       totalLatency / time.Duration(messageCount),
		P50Latency:       latencies[len(latencies)*50/100],
		P95Latency:       latencies[len(latencies)*95/100],
		P99Latency:       latencies[len(latencies)*99/100],
		BytesTransferred: bytesReceived.Load(),
		Throughput:       float64(bytesReceived.Load()) / (1024 * 1024) / duration.Seconds(),
	}

	return results
}

func BenchmarkJetStream(b *testing.B) {
	benchmarks := []struct {
		name         string
		messageCount int
		messageSize  int
		concurrent   int
	}{
		{"Small_1K_Sequential", 1000, 100, 1},
		{"Small_10K_Sequential", 10000, 100, 1},
		{"Small_10K_Concurrent", 10000, 100, 10},
		{"Medium_1K_Sequential", 1000, 1024, 1},
		{"Medium_10K_Concurrent", 10000, 1024, 10},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
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

			// Create stream
			_, err = js.CreateStream(context.Background(), jetstream.StreamConfig{
				Name:     "BENCH_STREAM",
				Subjects: []string{"bench.js.>"},
			})
			if err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				results := benchmarkJetStream(srv.ClientURL(), bm.messageCount, bm.messageSize, bm.concurrent)
				_ = results
			}
		})
	}
}

func benchmarkJetStream(natsURL string, messageCount, messageSize, concurrent int) *BenchmarkResults {
	subject := "bench.js.test"
	received := make(chan time.Duration, messageCount)
	var receivedCount atomic.Int32
	var bytesReceived atomic.Int64

	// Consumer configuration
	consumerConfig := fmt.Sprintf(`
input:
  nats_jetstream:
    urls: ["%s"]
    subject: "%s"
    deliver: "new"
    durable: "bench-consumer"
    ack_wait: "30s"
    max_ack_pending: %d

output:
  drop: {}
`, natsURL, subject, concurrent*100)

	// Create consumer
	consumerBuilder := service.NewStreamBuilder()
	if err := consumerBuilder.SetYAML(consumerConfig); err != nil {
		panic(err)
	}

	consumer, err := consumerBuilder.Build()
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consumer
	go func() {
		if err := consumer.Run(ctx); err != nil && ctx.Err() == nil {
			panic(err)
		}
	}()

	// Setup message tracking
	nc, err := nats.Connect(natsURL)
	if err != nil {
		panic(err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		panic(err)
	}

	// Create tracking consumer
	jsConsumer, err := js.CreateOrUpdateConsumer(context.Background(), "BENCH_STREAM", jetstream.ConsumerConfig{
		Durable:       "track-consumer",
		FilterSubject: subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		panic(err)
	}

	startTime := time.Now()

	// Subscribe to track messages
	cons, err := jsConsumer.Consume(func(msg jetstream.Msg) {
		receivedCount.Add(1)
		bytesReceived.Add(int64(len(msg.Data())))

		// Calculate latency from message metadata
		meta, _ := msg.Metadata()
		latency := time.Since(meta.Timestamp)
		received <- latency

		// Best effort ack - ignore errors in benchmark
		_ = msg.Ack()
	})
	if err != nil {
		panic(err)
	}
	defer cons.Stop()

	// Producer configuration
	producerConfig := fmt.Sprintf(`
input:
  generate:
    count: %d
    interval: "0s"
    mapping: |
      root.data = "test_payload_" + counter().string()
      root.size = %d

output:
  nats_jetstream:
    urls: ["%s"]
    subject: "%s"
    max_in_flight: %d
`, messageCount, messageSize, natsURL, subject, concurrent)

	// Create and run producer
	producerBuilder := service.NewStreamBuilder()
	if err := producerBuilder.SetYAML(producerConfig); err != nil {
		panic(err)
	}

	producer, err := producerBuilder.Build()
	if err != nil {
		panic(err)
	}

	if err := producer.Run(context.Background()); err != nil {
		panic(err)
	}

	// Wait for all messages
	timeout := time.After(30 * time.Second)
	for receivedCount.Load() < int32(messageCount) {
		select {
		case <-timeout:
			panic(fmt.Sprintf("timeout: received only %d/%d messages", receivedCount.Load(), messageCount))
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	duration := time.Since(startTime)
	close(received)

	// Calculate latencies
	latencies := make([]time.Duration, 0, messageCount)
	var totalLatency time.Duration
	for lat := range received {
		latencies = append(latencies, lat)
		totalLatency += lat
	}

	// Sort latencies for percentile calculation
	sortDurations(latencies)

	results := &BenchmarkResults{
		MessageCount:     messageCount,
		Duration:         duration,
		MessagesPerSec:   float64(messageCount) / duration.Seconds(),
		AvgLatency:       totalLatency / time.Duration(messageCount),
		P50Latency:       latencies[len(latencies)*50/100],
		P95Latency:       latencies[len(latencies)*95/100],
		P99Latency:       latencies[len(latencies)*99/100],
		BytesTransferred: bytesReceived.Load(),
		Throughput:       float64(bytesReceived.Load()) / (1024 * 1024) / duration.Seconds(),
	}

	return results
}

func BenchmarkJetStreamKV(b *testing.B) {
	benchmarks := []struct {
		name       string
		keyCount   int
		valueSize  int
		concurrent int
	}{
		{"Small_100_Sequential", 100, 100, 1},
		{"Small_1K_Sequential", 1000, 100, 1},
		{"Small_1K_Concurrent", 1000, 100, 10},
		{"Medium_100_Sequential", 100, 1024, 1},
		{"Medium_1K_Concurrent", 1000, 1024, 10},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
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

			// Create KV bucket
			_, err = js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
				Bucket: "BENCH_KV",
			})
			if err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				results := benchmarkJetStreamKV(srv.ClientURL(), bm.keyCount, bm.valueSize, bm.concurrent)
				_ = results
			}
		})
	}
}

func benchmarkJetStreamKV(natsURL string, keyCount, valueSize, concurrent int) *BenchmarkResults {
	startTime := time.Now()
	var bytesWritten atomic.Int64

	// Producer configuration
	producerConfig := fmt.Sprintf(`
input:
  generate:
    count: %d
    interval: "0s"
    mapping: |
      root.key = "bench_key_" + counter().string()
      root.value = "test_value_" + counter().string()

output:
  nats_kv:
    urls: ["%s"]
    bucket: "BENCH_KV"
    key: "${! json(\"key\") }"
    max_in_flight: %d
`, keyCount, natsURL, concurrent)

	// Create and run producer
	builder := service.NewStreamBuilder()
	if err := builder.SetYAML(producerConfig); err != nil {
		panic(err)
	}

	stream, err := builder.Build()
	if err != nil {
		panic(err)
	}

	// Track bytes written
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		nc, _ := nats.Connect(natsURL)
		defer nc.Close()
		js, _ := jetstream.New(nc)
		kv, _ := js.KeyValue(context.Background(), "BENCH_KV")

		for i := 1; i <= keyCount; i++ {
			key := fmt.Sprintf("bench_key_%d", i)
			if entry, err := kv.Get(context.Background(), key); err == nil {
				bytesWritten.Add(int64(len(entry.Value())))
			}
		}
	}()

	if err := stream.Run(context.Background()); err != nil {
		panic(err)
	}

	wg.Wait()
	duration := time.Since(startTime)

	results := &BenchmarkResults{
		MessageCount:     keyCount,
		Duration:         duration,
		MessagesPerSec:   float64(keyCount) / duration.Seconds(),
		BytesTransferred: bytesWritten.Load(),
		Throughput:       float64(bytesWritten.Load()) / (1024 * 1024) / duration.Seconds(),
	}

	return results
}

// Helper function to sort durations
func sortDurations(durations []time.Duration) {
	for i := 0; i < len(durations); i++ {
		for j := i + 1; j < len(durations); j++ {
			if durations[i] > durations[j] {
				durations[i], durations[j] = durations[j], durations[i]
			}
		}
	}
}
