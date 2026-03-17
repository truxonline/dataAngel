module github.com/charchess/dataAngel/sidecar-litestream

go 1.22.2

replace github.com/charchess/dataAngel/internal/k8s => ../../internal/k8s

require github.com/charchess/dataAngel/internal/k8s v0.0.0-00010101000000-000000000000
