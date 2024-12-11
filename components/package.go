package components

import (
	_ "github.com/redpanda-data/benthos/v4/public/components/io"
	_ "github.com/redpanda-data/benthos/v4/public/components/pure"
	_ "github.com/redpanda-data/connect/v4/public/components/aws"
	_ "github.com/redpanda-data/connect/v4/public/components/azure"
	_ "github.com/redpanda-data/connect/v4/public/components/changelog"
	_ "github.com/redpanda-data/connect/v4/public/components/cockroachdb"
	_ "github.com/redpanda-data/connect/v4/public/components/gcp"
	_ "github.com/redpanda-data/connect/v4/public/components/io"
	_ "github.com/redpanda-data/connect/v4/public/components/nats"
	_ "github.com/redpanda-data/connect/v4/public/components/prometheus"
	_ "github.com/redpanda-data/connect/v4/public/components/sql"
	_ "github.com/wombatwisdom/wombat-plugins/components/nats"
)
