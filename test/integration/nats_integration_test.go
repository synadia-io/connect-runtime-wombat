package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redpanda-data/benthos/v4/public/service"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	// Import all components
	_ "github.com/synadia-io/connect-runtime-wombat/components"
)

var _ = Describe("NATS Integration Tests", func() {
	var (
		srv *server.Server
		nc  *nats.Conn
		js  jetstream.JetStream
	)

	BeforeEach(func() {
		// Start NATS server with JetStream
		opts := test.DefaultTestOptions
		opts.Port = -1
		opts.JetStream = true
		opts.StoreDir = GinkgoT().TempDir()

		var err error
		srv = test.RunServer(&opts)
		Expect(srv).NotTo(BeNil())

		// Connect to server
		nc, err = nats.Connect(srv.ClientURL())
		Expect(err).NotTo(HaveOccurred())

		// Create JetStream context
		js, err = jetstream.New(nc)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if nc != nil {
			nc.Close()
		}
		if srv != nil {
			srv.Shutdown()
		}
	})

	Describe("Core NATS", func() {
		It("should produce and consume messages", func() {
			subject := "test.core.subject"
			messageCount := 100
			received := make([]string, 0, messageCount)
			var mu sync.Mutex

			// Create consumer
			consumerConfig := fmt.Sprintf(`
input:
  nats:
    urls: ["%s"]
    subject: "%s"
    queue: "test-queue"

output:
  drop: {}
`, srv.ClientURL(), subject)

			consumerBuilder := service.NewStreamBuilder()
			Expect(consumerBuilder.SetYAML(consumerConfig)).To(Succeed())

			consumer, err := consumerBuilder.Build()
			Expect(err).NotTo(HaveOccurred())

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Start consumer
			go func() {
				defer GinkgoRecover()
				err := consumer.Run(ctx)
				if err != nil && ctx.Err() == nil {
					Expect(err).NotTo(HaveOccurred())
				}
			}()

			// Subscribe to verify messages
			sub, err := nc.QueueSubscribe(subject, "verify-queue", func(msg *nats.Msg) {
				mu.Lock()
				defer mu.Unlock()
				received = append(received, string(msg.Data))
			})
			Expect(err).NotTo(HaveOccurred())
			defer sub.Unsubscribe()

			// Create producer
			producerConfig := fmt.Sprintf(`
input:
  generate:
    count: %d
    interval: "1ms"
    mapping: |
      root.id = counter()
      root.timestamp = now()
      root.data = "test message " + counter().string()

output:
  nats:
    urls: ["%s"]
    subject: "%s"
`, messageCount, srv.ClientURL(), subject)

			producerBuilder := service.NewStreamBuilder()
			Expect(producerBuilder.SetYAML(producerConfig)).To(Succeed())

			producer, err := producerBuilder.Build()
			Expect(err).NotTo(HaveOccurred())

			// Run producer
			err = producer.Run(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Wait for messages
			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(received)
			}, 5*time.Second, 100*time.Millisecond).Should(Equal(messageCount))

			// Verify message content
			mu.Lock()
			receivedIDs := make(map[float64]bool)
			for _, msg := range received {
				var data map[string]interface{}
				Expect(json.Unmarshal([]byte(msg), &data)).To(Succeed())
				id := data["id"].(float64)
				Expect(id).To(BeNumerically(">=", 1))
				Expect(id).To(BeNumerically("<=", float64(messageCount)))
				expectedData := fmt.Sprintf("test message %d", int(id))
				Expect(data["data"]).To(Equal(expectedData))
				receivedIDs[id] = true
			}
			// Verify we received all unique IDs
			Expect(len(receivedIDs)).To(Equal(messageCount))
			mu.Unlock()
		})
	})

	Describe("JetStream", func() {
		var streamName string

		BeforeEach(func() {
			streamName = "TEST_STREAM"
			// Create stream
			_, err := js.CreateStream(context.Background(), jetstream.StreamConfig{
				Name:     streamName,
				Subjects: []string{"test.js.>"},
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should produce and consume messages from JetStream", func() {
			subject := "test.js.messages"
			messageCount := 100
			var received atomic.Int32

			// Create consumer
			consumerConfig := fmt.Sprintf(`
input:
  nats_jetstream:
    urls: ["%s"]
    subject: "%s"
    deliver: "all"
    durable: "test-consumer"
    ack_wait: "5s"
    max_ack_pending: 1024

output:
  drop: {}
`, srv.ClientURL(), subject)

			consumerBuilder := service.NewStreamBuilder()
			Expect(consumerBuilder.SetYAML(consumerConfig)).To(Succeed())

			consumer, err := consumerBuilder.Build()
			Expect(err).NotTo(HaveOccurred())

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Start consumer
			go func() {
				defer GinkgoRecover()
				err := consumer.Run(ctx)
				if err != nil && ctx.Err() == nil {
					Expect(err).NotTo(HaveOccurred())
				}
			}()

			// Create JetStream consumer to count messages
			jsConsumer, err := js.CreateOrUpdateConsumer(context.Background(), streamName, jetstream.ConsumerConfig{
				Durable:       "verify-consumer",
				FilterSubject: subject,
				AckPolicy:     jetstream.AckExplicitPolicy,
			})
			Expect(err).NotTo(HaveOccurred())

			// Subscribe to count messages
			msgChan := make(chan jetstream.Msg, messageCount)
			var msgChanMu sync.Mutex
			var msgChanClosed bool
			cons, err := jsConsumer.Consume(func(msg jetstream.Msg) {
				received.Add(1)
				msgChanMu.Lock()
				if !msgChanClosed {
					msgChan <- msg
				}
				msgChanMu.Unlock()
			})
			Expect(err).NotTo(HaveOccurred())
			defer cons.Stop()

			// Create producer
			producerConfig := fmt.Sprintf(`
input:
  generate:
    count: %d
    interval: "1ms"
    mapping: |
      root.id = counter()
      root.timestamp = now()
      root.data = "jetstream message " + counter().string()

output:
  nats_jetstream:
    urls: ["%s"]
    subject: "%s"
    max_in_flight: 10
`, messageCount, srv.ClientURL(), subject)

			producerBuilder := service.NewStreamBuilder()
			Expect(producerBuilder.SetYAML(producerConfig)).To(Succeed())

			producer, err := producerBuilder.Build()
			Expect(err).NotTo(HaveOccurred())

			// Run producer
			err = producer.Run(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Wait for messages
			Eventually(func() int32 {
				return received.Load()
			}, 10*time.Second, 100*time.Millisecond).Should(Equal(int32(messageCount)))

			// Acknowledge messages
			msgChanMu.Lock()
			msgChanClosed = true
			close(msgChan)
			msgChanMu.Unlock()
			for msg := range msgChan {
				Expect(msg.Ack()).To(Succeed())
			}
		})
	})

	Describe("JetStream KV", func() {
		var kvBucket string

		BeforeEach(func() {
			kvBucket = "TEST_KV_BUCKET"
			// Create KV bucket
			_, err := js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
				Bucket: kvBucket,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should write and read from KV store", func() {
			messageCount := 50

			// Create producer to write to KV
			producerConfig := fmt.Sprintf(`
input:
  generate:
    count: %d
    interval: "1ms"
    mapping: |
      root.key = "item_" + counter().string()
      root.value = {
        "id": counter(),
        "timestamp": now(),
        "data": "kv data " + counter().string()
      }

output:
  nats_kv:
    urls: ["%s"]
    bucket: "%s"
    key: "${! json(\"key\") }"
`, messageCount, srv.ClientURL(), kvBucket)

			producerBuilder := service.NewStreamBuilder()
			Expect(producerBuilder.SetYAML(producerConfig)).To(Succeed())

			producer, err := producerBuilder.Build()
			Expect(err).NotTo(HaveOccurred())

			// Run producer
			err = producer.Run(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Verify data in KV
			kv, err := js.KeyValue(context.Background(), kvBucket)
			Expect(err).NotTo(HaveOccurred())

			// Check all keys exist
			for i := 1; i <= messageCount; i++ {
				key := fmt.Sprintf("item_%d", i)
				entry, err := kv.Get(context.Background(), key)
				Expect(err).NotTo(HaveOccurred())

				var data map[string]interface{}
				Expect(json.Unmarshal(entry.Value(), &data)).To(Succeed())

				value := data["value"].(map[string]interface{})
				Expect(value["id"]).To(Equal(float64(i)))
				Expect(value["data"]).To(Equal(fmt.Sprintf("kv data %d", i)))
			}

			// Create consumer to read from KV
			consumerConfig := fmt.Sprintf(`
input:
  nats_kv:
    urls: ["%s"]
    bucket: "%s"
    key: "item_25"

output:
  drop: {}
`, srv.ClientURL(), kvBucket)

			consumerBuilder := service.NewStreamBuilder()
			Expect(consumerBuilder.SetYAML(consumerConfig)).To(Succeed())

			// Test that config is valid
			consumer, err := consumerBuilder.Build()
			Expect(err).NotTo(HaveOccurred())
			
			// We don't run the consumer as it would block waiting for updates
			// Just verify it can be built successfully
			_ = consumer
		})

		It("should watch KV updates", func() {
			key := "watch_key"
			updates := make([]string, 0)
			var mu sync.Mutex

			// Get KV handle
			kv, err := js.KeyValue(context.Background(), kvBucket)
			Expect(err).NotTo(HaveOccurred())

			// Watch for updates
			watcher, err := kv.Watch(context.Background(), key)
			Expect(err).NotTo(HaveOccurred())
			defer watcher.Stop()

			go func() {
				for entry := range watcher.Updates() {
					if entry != nil {
						mu.Lock()
						updates = append(updates, string(entry.Value()))
						mu.Unlock()
					}
				}
			}()

			// Write updates using producer
			updateCount := 10
			for i := 1; i <= updateCount; i++ {
				producerConfig := fmt.Sprintf(`
input:
  generate:
    count: 1
    mapping: |
      root = "update_%d"

output:
  nats_kv:
    urls: ["%s"]
    bucket: "%s"
    key: "%s"
`, i, srv.ClientURL(), kvBucket, key)

				producerBuilder := service.NewStreamBuilder()
				Expect(producerBuilder.SetYAML(producerConfig)).To(Succeed())

				producer, err := producerBuilder.Build()
				Expect(err).NotTo(HaveOccurred())

				err = producer.Run(context.Background())
				Expect(err).NotTo(HaveOccurred())

				time.Sleep(10 * time.Millisecond) // Small delay between updates
			}

			// Verify all updates received
			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(updates)
			}, 5*time.Second, 100*time.Millisecond).Should(Equal(updateCount))

			// Verify update content
			mu.Lock()
			for i, update := range updates {
				Expect(update).To(Equal(fmt.Sprintf("update_%d", i+1)))
			}
			mu.Unlock()
		})
	})
})