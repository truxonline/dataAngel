module github.com/charchess/dataAngel/cmd/dataangel

go 1.23.0

replace github.com/charchess/dataAngel/internal/sidecar => ../../internal/sidecar

require (
	github.com/charchess/dataAngel/internal/sidecar v0.0.0-00010101000000-000000000000
	github.com/mattn/go-sqlite3 v1.14.37
	github.com/prometheus/client_golang v1.23.2
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
