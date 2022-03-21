module github.com/DataDog/datadog-agent/pkg/otlp/model

go 1.16

replace github.com/DataDog/datadog-agent/pkg/quantile => ../../quantile

require (
	github.com/DataDog/datadog-agent/pkg/quantile v0.32.0-rc.5
	github.com/kr/text v0.2.0 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/collector/model v0.47.0
	go.uber.org/zap v1.19.1
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)
