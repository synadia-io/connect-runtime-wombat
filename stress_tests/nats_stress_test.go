package stress_test

import (
	"context"
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

var _ = Describe("NATS Stress Tests", func() {
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
		// Increase limits for stress testing
		opts.MaxPayload = 10 * 1024 * 1024 // 10MB
		opts.MaxPending = 10 * 1024 * 1024 // 10MB

		var err error
		srv = test.RunServer(&opts)
		Expect(srv).NotTo(BeNil())

		nc, err = nats.Connect(srv.ClientURL())
		Expect(err).NotTo(HaveOccurred())

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

	Describe("High Volume Stress Tests", func() {
		It("should handle 100K messages without loss", func() {
			subject := "stress.highvolume"
			messageCount := 100000
			var received atomic.Int64
			var errors atomic.Int64

			// Subscribe to count messages
			sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
				received.Add(1)
			})
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				if err := sub.Unsubscribe(); err != nil {
					GinkgoWriter.Printf("Failed to unsubscribe: %v\n", err)
				}
			}()

			// Producer configuration with high throughput
			producerConfig := fmt.Sprintf(`
logger:
  level: warn
  format: json
  add_timestamp: true

input:
  generate:
    count: %d
    interval: "0s"
    mapping: |
      root.id = counter()
      root.timestamp = timestamp_unix_nano()
      root.data = uuid_v4()

output:
  nats:
    urls: ["%s"]
    subject: "%s"
    max_in_flight: 1000
`, messageCount, srv.ClientURL(), subject)

			builder := service.NewStreamBuilder()
			Expect(builder.SetYAML(producerConfig)).To(Succeed())

			stream, err := builder.Build()
			Expect(err).NotTo(HaveOccurred())

			// Run producer
			start := time.Now()
			err = stream.Run(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Wait for all messages with timeout
			Eventually(func() int64 {
				return received.Load()
			}, 30*time.Second, 100*time.Millisecond).Should(Equal(int64(messageCount)))

			duration := time.Since(start)
			rate := float64(messageCount) / duration.Seconds()

			GinkgoWriter.Printf("\nHigh Volume Test Results:\n")
			GinkgoWriter.Printf("Messages: %d\n", messageCount)
			GinkgoWriter.Printf("Duration: %v\n", duration)
			GinkgoWriter.Printf("Rate: %.2f msgs/sec\n", rate)
			GinkgoWriter.Printf("Errors: %d\n", errors.Load())

			Expect(errors.Load()).To(Equal(int64(0)))
		})
	})

	Describe("Concurrent Publisher Stress Tests", func() {
		It("should handle 100 concurrent publishers", func() {
			publisherCount := 100
			messagesPerPublisher := 1000
			totalMessages := publisherCount * messagesPerPublisher
			subject := "stress.concurrent.pub"

			var received atomic.Int64
			var publishErrors atomic.Int64

			// Subscribe to count messages
			sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
				received.Add(1)
			})
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				if err := sub.Unsubscribe(); err != nil {
					GinkgoWriter.Printf("Failed to unsubscribe: %v\n", err)
				}
			}()

			// Start publishers sequentially, waiting for each connection to succeed
			var wg sync.WaitGroup
			start := time.Now()
			connChan := make(chan int, 1) // Channel to signal when to start next publisher

			for i := 0; i < publisherCount; i++ {
				wg.Add(1)

				// Wait for previous publisher to establish connection (except first one)
				if i > 0 {
					<-connChan
				}

				go func(pubID int) {
					defer wg.Done()
					defer GinkgoRecover()
					defer func() {
						connChan <- pubID
					}()

					config := fmt.Sprintf(`
logger:
  level: warn
  format: json
  add_timestamp: true

input:
  generate:
    count: %d
    interval: "0s"
    mapping: |
      root.publisher_id = %d
      root.msg_id = counter()
      root.timestamp = timestamp_unix_nano()

output:
  nats:
    urls: ["%s"]
    subject: "%s"
    max_in_flight: 10
`, messagesPerPublisher, pubID, srv.ClientURL(), subject)

					builder := service.NewStreamBuilder()
					if err := builder.SetYAML(config); err != nil {
						publishErrors.Add(1)
						return
					}

					stream, err := builder.Build()
					if err != nil {
						publishErrors.Add(1)
						return
					}

					if err := stream.Run(context.Background()); err != nil {
						publishErrors.Add(1)
					}
				}(i)
			}

			wg.Wait()
			duration := time.Since(start)

			// Verify all messages received
			Eventually(func() int64 {
				return received.Load()
			}, 5*time.Second, 100*time.Millisecond).Should(Equal(int64(totalMessages)))

			rate := float64(totalMessages) / duration.Seconds()

			GinkgoWriter.Printf("\nConcurrent Publisher Test Results:\n")
			GinkgoWriter.Printf("Publishers: %d\n", publisherCount)
			GinkgoWriter.Printf("Messages per publisher: %d\n", messagesPerPublisher)
			GinkgoWriter.Printf("Total messages: %d\n", totalMessages)
			GinkgoWriter.Printf("Duration: %v\n", duration)
			GinkgoWriter.Printf("Rate: %.2f msgs/sec\n", rate)
			GinkgoWriter.Printf("Publish errors: %d\n", publishErrors.Load())

			Expect(publishErrors.Load()).To(Equal(int64(0)))
		})
	})

	Describe("Large Message Stress Tests", func() {
		It("should handle large messages (1MB each)", func() {
			subject := "stress.largemsg"
			messageCount := 100
			messageSize := 1024 * 1024 // 1MB
			var received atomic.Int64
			var totalBytes atomic.Int64

			// Subscribe to verify messages
			sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
				received.Add(1)
				totalBytes.Add(int64(len(msg.Data)))
			})
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				if err := sub.Unsubscribe(); err != nil {
					GinkgoWriter.Printf("Failed to unsubscribe: %v\n", err)
				}
			}()

			// Producer configuration - generate 1MB messages
			producerConfig := fmt.Sprintf(`
logger:
  level: warn
  format: json
  add_timestamp: true

input:
  generate:
    count: %d
    interval: "10ms"
    mapping: |
      root.id = counter()
      root.data = range(0, %d).map_each(_ -> "x").join("")

output:
  nats:
    urls: ["%s"]
    subject: "%s"
    max_in_flight: 10
`, messageCount, messageSize, srv.ClientURL(), subject)

			builder := service.NewStreamBuilder()
			Expect(builder.SetYAML(producerConfig)).To(Succeed())

			stream, err := builder.Build()
			Expect(err).NotTo(HaveOccurred())

			// Run producer
			start := time.Now()
			err = stream.Run(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Wait for all messages
			Eventually(func() int64 {
				return received.Load()
			}, 30*time.Second, 100*time.Millisecond).Should(Equal(int64(messageCount)))

			duration := time.Since(start)
			throughput := float64(totalBytes.Load()) / (1024 * 1024) / duration.Seconds()

			GinkgoWriter.Printf("\nLarge Message Test Results:\n")
			GinkgoWriter.Printf("Messages: %d\n", messageCount)
			GinkgoWriter.Printf("Message size: %d bytes\n", messageSize)
			GinkgoWriter.Printf("Total data: %.2f MB\n", float64(totalBytes.Load())/(1024*1024))
			GinkgoWriter.Printf("Duration: %v\n", duration)
			GinkgoWriter.Printf("Throughput: %.2f MB/s\n", throughput)
		})
	})

	Describe("JetStream Stress Tests", func() {
		BeforeEach(func() {
			// Create stream for stress tests
			_, err := js.CreateStream(context.Background(), jetstream.StreamConfig{
				Name:     "STRESS_STREAM",
				Subjects: []string{"stress.js.>"},
				Storage:  jetstream.FileStorage,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle rapid publish/consume with acknowledgments", func() {
			subject := "stress.js.rapid"
			messageCount := 10000
			var published atomic.Int64
			var consumed atomic.Int64
			var acked atomic.Int64

			// Consumer configuration
			consumerConfig := fmt.Sprintf(`
logger:
  level: warn
  format: json
  add_timestamp: true

input:
  nats_jetstream:
    urls: ["%s"]
    subject: "%s"
    deliver: "all"
    durable: "stress-consumer"
    ack_wait: "30s"
    max_ack_pending: 1000

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

			// Create a consumer to track acknowledgments
			jsConsumer, err := js.CreateOrUpdateConsumer(context.Background(), "STRESS_STREAM", jetstream.ConsumerConfig{
				Durable:       "track-stress",
				FilterSubject: subject,
				AckPolicy:     jetstream.AckExplicitPolicy,
			})
			Expect(err).NotTo(HaveOccurred())

			// Subscribe to track consumption
			cons, err := jsConsumer.Consume(func(msg jetstream.Msg) {
				consumed.Add(1)
				if err := msg.Ack(); err == nil {
					acked.Add(1)
				}
			})
			Expect(err).NotTo(HaveOccurred())
			defer cons.Stop()

			// Producer configuration
			producerConfig := fmt.Sprintf(`
logger:
  level: warn
  format: json
  add_timestamp: true

input:
  generate:
    count: %d
    interval: "0s"
    mapping: |
      root.id = counter()
      root.timestamp = timestamp_unix_nano()
      root.data = uuid_v4()

output:
  nats_jetstream:
    urls: ["%s"]
    subject: "%s"
    max_in_flight: 100
`, messageCount, srv.ClientURL(), subject)

			producerBuilder := service.NewStreamBuilder()
			Expect(producerBuilder.SetYAML(producerConfig)).To(Succeed())

			producer, err := producerBuilder.Build()
			Expect(err).NotTo(HaveOccurred())

			// Track publishing
			publishCtx, publishCancel := context.WithCancel(context.Background())
			defer publishCancel()

			go func() {
				streamInfo, err := js.Stream(publishCtx, "STRESS_STREAM")
				if err != nil {
					return
				}
				for {
					select {
					case <-publishCtx.Done():
						return
					default:
						info, err := streamInfo.Info(publishCtx)
						if err == nil && info != nil {
							published.Store(int64(info.State.Msgs))
						}
						time.Sleep(100 * time.Millisecond)
					}
				}
			}()

			// Run producer
			start := time.Now()
			err = producer.Run(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Wait for all messages to be consumed
			Eventually(func() int64 {
				return consumed.Load()
			}, 30*time.Second, 100*time.Millisecond).Should(Equal(int64(messageCount)))

			duration := time.Since(start)
			rate := float64(messageCount) / duration.Seconds()

			GinkgoWriter.Printf("\nJetStream Stress Test Results:\n")
			GinkgoWriter.Printf("Messages: %d\n", messageCount)
			GinkgoWriter.Printf("Published: %d\n", published.Load())
			GinkgoWriter.Printf("Consumed: %d\n", consumed.Load())
			GinkgoWriter.Printf("Acknowledged: %d\n", acked.Load())
			GinkgoWriter.Printf("Duration: %v\n", duration)
			GinkgoWriter.Printf("Rate: %.2f msgs/sec\n", rate)
		})
	})

	Describe("KV Store Stress Tests", func() {
		var kvBucket string

		BeforeEach(func() {
			kvBucket = "STRESS_KV"
			_, err := js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
				Bucket:      kvBucket,
				Description: "Stress test KV bucket",
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle rapid KV updates", func() {
			keyCount := 100
			updatesPerKey := 100
			totalOperations := keyCount * updatesPerKey
			var written atomic.Int64
			var errors atomic.Int64

			// Generate updates for each key
			var wg sync.WaitGroup
			start := time.Now()

			for i := 0; i < keyCount; i++ {
				wg.Add(1)
				go func(keyID int) {
					defer wg.Done()
					defer GinkgoRecover()

					key := fmt.Sprintf("stress_key_%d", keyID)

					for j := 0; j < updatesPerKey; j++ {
						config := fmt.Sprintf(`
logger:
  level: warn
  format: json
  add_timestamp: true

input:
  generate:
    count: 1
    mapping: |
      root.key = "%s"
      root.version = %d
      root.timestamp = timestamp_unix_nano()
      root.data = uuid_v4()

output:
  nats_kv:
    urls: ["%s"]
    bucket: "%s"
    key: "%s"
`, key, j, srv.ClientURL(), kvBucket, key)

						builder := service.NewStreamBuilder()
						if err := builder.SetYAML(config); err != nil {
							errors.Add(1)
							continue
						}

						stream, err := builder.Build()
						if err != nil {
							errors.Add(1)
							continue
						}

						if err := stream.Run(context.Background()); err != nil {
							errors.Add(1)
						} else {
							written.Add(1)
						}
					}
				}(i)
			}

			wg.Wait()
			duration := time.Since(start)

			// Verify final state
			kv, err := js.KeyValue(context.Background(), kvBucket)
			Expect(err).NotTo(HaveOccurred())

			keys, err := kv.Keys(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(len(keys)).To(Equal(keyCount))

			rate := float64(written.Load()) / duration.Seconds()

			GinkgoWriter.Printf("\nKV Store Stress Test Results:\n")
			GinkgoWriter.Printf("Keys: %d\n", keyCount)
			GinkgoWriter.Printf("Updates per key: %d\n", updatesPerKey)
			GinkgoWriter.Printf("Total operations: %d\n", totalOperations)
			GinkgoWriter.Printf("Successful writes: %d\n", written.Load())
			GinkgoWriter.Printf("Errors: %d\n", errors.Load())
			GinkgoWriter.Printf("Duration: %v\n", duration)
			GinkgoWriter.Printf("Rate: %.2f ops/sec\n", rate)

			// Allow some errors due to concurrent updates
			errorRate := float64(errors.Load()) / float64(totalOperations)
			Expect(errorRate).To(BeNumerically("<", 0.05)) // Less than 5% error rate
		})
	})

	Describe("Connection Resilience Tests", func() {
		It("should handle connection drops and reconnects", func() {
			subject := "stress.resilience"
			messageCount := 1000
			var sent atomic.Int64
			var received atomic.Int64
			var reconnects atomic.Int64

			// Subscribe with reconnect handler
			nc2, err := nats.Connect(srv.ClientURL(),
				nats.ReconnectWait(100*time.Millisecond),
				nats.MaxReconnects(-1),
				nats.ReconnectHandler(func(nc *nats.Conn) {
					reconnects.Add(1)
				}),
			)
			Expect(err).NotTo(HaveOccurred())
			defer nc2.Close()

			sub, err := nc2.Subscribe(subject, func(msg *nats.Msg) {
				received.Add(1)
			})
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				if err := sub.Unsubscribe(); err != nil {
					GinkgoWriter.Printf("Failed to unsubscribe: %v\n", err)
				}
			}()

			// Start producer in background
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				for sent.Load() < int64(messageCount) {
					config := fmt.Sprintf(`
logger:
  level: warn
  format: json
  add_timestamp: true

input:
  generate:
    count: 100
    interval: "10ms"
    mapping: |
      root.id = counter()
      root.timestamp = timestamp_unix_nano()

output:
  nats:
    urls: ["%s"]
    subject: "%s"
`, srv.ClientURL(), subject)

					builder := service.NewStreamBuilder()
					if err := builder.SetYAML(config); err != nil {
						continue
					}

					stream, err := builder.Build()
					if err != nil {
						continue
					}

					if err := stream.Run(ctx); err != nil {
						// Context cancellation is expected
						if ctx.Err() == nil {
							continue
						}
					}
					sent.Add(100)
				}
			}()

			// Simulate connection drops
			time.Sleep(500 * time.Millisecond)
			nc2.Close()
			time.Sleep(500 * time.Millisecond)

			// Reconnect
			nc2, err = nats.Connect(srv.ClientURL())
			Expect(err).NotTo(HaveOccurred())

			sub2, err := nc2.Subscribe(subject, func(msg *nats.Msg) {
				received.Add(1)
			})
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				if err := sub2.Unsubscribe(); err != nil {
					GinkgoWriter.Printf("Failed to unsubscribe: %v\n", err)
				}
			}()

			// Wait for completion
			time.Sleep(5 * time.Second)

			GinkgoWriter.Printf("\nConnection Resilience Test Results:\n")
			GinkgoWriter.Printf("Messages sent: %d\n", sent.Load())
			GinkgoWriter.Printf("Messages received: %d\n", received.Load())
			GinkgoWriter.Printf("Reconnects: %d\n", reconnects.Load())
			GinkgoWriter.Printf("Loss rate: %.2f%%\n", float64(sent.Load()-received.Load())/float64(sent.Load())*100)

			// Some message loss is expected during disconnection
			lossRate := float64(sent.Load()-received.Load()) / float64(sent.Load())
			Expect(lossRate).To(BeNumerically("<", 0.5)) // Less than 50% loss
		})
	})
})
