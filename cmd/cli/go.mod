module github.com/charchess/dataAngel/cmd/cli

go 1.22.2

replace github.com/charchess/dataAngel/pkg/s3 => ../../pkg/s3

require github.com/charchess/dataAngel/pkg/s3 v0.0.0-00010101000000-000000000000
