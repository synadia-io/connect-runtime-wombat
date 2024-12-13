package compiler

import (
	"github.com/rs/zerolog/log"
	"github.com/synadia-io/connect/model"
)

func compileProducer(steps model.Steps) (map[string]any, error) {
	var result map[string]any
	var err error

	if steps.Producer.JetStream != nil {
		result, err = compileJetStreamProducer(steps)
	} else {
		result, err = compileCoreProducer(steps)
	}

	if err != nil {
		return nil, err
	}

	if steps.Transformer != nil {
		return attachTransformerAsProcessor(result, steps)
	}

	return result, nil
}

func compileCoreProducer(steps model.Steps) (map[string]any, error) {
	cfg := map[string]any{
		"subject":       steps.Producer.Subject,
		"max_in_flight": 1,
		"metadata": map[string]any{
			"include_patterns": []string{
				".*",
			},
		},
	}

	attachNatsConfig(cfg, steps.Producer.NatsConfig)

	if steps.Producer.Threads > 0 {
		cfg["max_in_flight"] = steps.Producer.Threads
	}

	return map[string]any{"nats": cfg}, nil
}

func compileJetStreamProducer(steps model.Steps) (map[string]any, error) {
	cfg := map[string]any{
		"subject":       steps.Producer.Subject,
		"max_in_flight": 1,
	}

	attachNatsConfig(cfg, steps.Producer.NatsConfig)

	if steps.Producer.Threads > 0 {
		cfg["max_in_flight"] = steps.Producer.Threads
	}

	if steps.Producer.JetStream.MsgId != "" {
		log.Warn().Msgf("msg_id is not supported for jetstream producer")
		//cfg["msg_id"] = v.Producer.JetStream.MsgId
	}

	if steps.Producer.JetStream.AckWait != "" {
		log.Warn().Msgf("ack_wait is not supported for jetstream producer")
		//cfg["ack_wait"] = v.Producer.JetStream.AckWait
	}

	if steps.Producer.JetStream.Batching != nil {
		log.Warn().Msgf("batching is not supported for jetstream producer")
		//cfg["batching"] = map[string]any{
		//	"count":     v.Producer.JetStream.Batching.Count,
		//	"byte_size": v.Producer.JetStream.Batching.ByteSize,
		//}
	}

	result := map[string]any{"nats_jetstream": cfg}

	return result, nil
}
