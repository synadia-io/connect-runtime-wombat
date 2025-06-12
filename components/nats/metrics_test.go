package nats_test

import (
	"bytes"
	"context"
	"fmt"
	"time"

	nats2 "github.com/nats-io/nats.go"
	"github.com/nats-io/nuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/common/expfmt"
	"github.com/redpanda-data/benthos/v4/public/service"

	_ "github.com/redpanda-data/benthos/v4/public/components/io"
	_ "github.com/redpanda-data/benthos/v4/public/components/pure"
	_ "github.com/redpanda-data/benthos/v4/public/components/pure/extended"
)

var _ = Describe("Metrics", func() {
	It("should send metrics to nats", func() {
		subject := fmt.Sprintf("metrics.%s", nuid.Next())
		cfg := fmt.Sprintf(`
input:
  generate:
    mapping: |-
      root = "Hello, world!"
    interval: 1s

output:
  drop: {}

metrics:
  nats:
    url: %s
    subject: %s
    flush_interval: 1s
`, srv.ClientURL(), subject)

		sb := service.NewStreamBuilder()
		Expect(sb.SetYAML(cfg)).To(BeNil())

		strm, err := sb.Build()
		Expect(err).To(BeNil())

		// -- start listening for metrics
		var messages []*nats2.Msg
		s, err := nc.Subscribe(subject, func(msg *nats2.Msg) {
			messages = append(messages, msg)
		})
		Expect(err).To(BeNil())

		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
		defer cancel()

		_ = strm.Run(ctx)

		// stop subscribing
		Expect(s.Drain()).To(BeNil())

		Expect(len(messages)).To(BeNumerically(">", 0))

		for _, msg := range messages {
			b := bytes.NewBuffer(msg.Data)
			tp := expfmt.TextParser{}
			fams, err := tp.TextToMetricFamilies(b)
			Expect(err).To(BeNil())

			Expect(fams).ToNot(BeEmpty())
		}

	})
})
