package compiler

import (
	"fmt"
	"github.com/synadia-io/connect/model"
)

func attachTransformerAsProcessor(result map[string]any, steps model.Steps) (map[string]any, error) {
	tm, err := compileTransformer(steps.Transformer)
	if err != nil {
		return nil, fmt.Errorf("transformer: %w", err)
	}

	if tm != nil {
		result["processors"] = []map[string]any{tm}
	}

	return result, nil
}

func compileTransformer(transformer *model.Transformer) (map[string]any, error) {
	if transformer == nil {
		return nil, nil
	}

	if transformer.Composite != nil {
		return compileCompositeTransformer(transformer.Composite)
	}

	if transformer.Service != nil {
		return compileServiceTransformer(transformer.Service)
	}

	if transformer.Mapping != nil {
		return compileMappingTransformer(transformer.Mapping)
	}

	return nil, nil
}

func compileServiceTransformer(t *model.ServiceTransformer) (map[string]any, error) {
	cfg := map[string]any{
		"urls":    []string{t.NatsConfig.Url},
		"subject": t.Endpoint,
		"metadata": map[string]any{
			"include_patterns": []string{
				".*",
			},
		},
	}

	if t.NatsConfig.AuthEnabled {
		cfg["auth"] = map[string]string{
			"user_jwt":       t.NatsConfig.Jwt,
			"user_nkey_seed": t.NatsConfig.Seed,
		}
	}

	if t.Timeout != "" {
		cfg["timeout"] = t.Timeout
	}

	return map[string]any{"nats_request_reply": cfg}, nil
}

func compileCompositeTransformer(t *model.CompositeTransformer) (map[string]any, error) {
	result := map[string]any{
		"processors": []map[string]any{},
	}

	for _, ct := range t.Sequential {
		compiled, err := compileTransformer(&ct)
		if err != nil {
			return nil, fmt.Errorf("composite transformer: %w", err)
		}

		if compiled != nil {
			result["processors"] = append(result["processors"].([]map[string]any), compiled)
		}
	}

	return result, nil
}

func compileMappingTransformer(t *model.MappingTransformer) (map[string]any, error) {
	return map[string]any{"mapping": t.Sourcecode}, nil
}
