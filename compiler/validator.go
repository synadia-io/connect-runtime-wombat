package compiler

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/rs/zerolog/log"
	_ "github.com/synadia-io/connect-runtime-wombat/components"
	"github.com/synadia-io/connect/runtime"
	"net/http"
)

func Validate(ctx context.Context, runtime *runtime.Runtime, code string, mux *http.ServeMux) (*service.Stream, error) {
	sb := service.NewStreamBuilder()
	sb.SetLogger(runtime.Logger)
	sb.SetHTTPMux(mux)

	if err := sb.SetYAML(code); err != nil {
		return nil, fmt.Errorf("invalid artifact: %w", err)
	}

	y, _ := sb.AsYAML()
	if y != "" {
		log.Info().Msgf("stream def: %s", base64.StdEncoding.EncodeToString([]byte(y)))
	}

	return sb.Build()
}
