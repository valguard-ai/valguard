module github.com/valguard-ai/valguard

go 1.25.0

require (
	github.com/valguard-ai/valguard/apps/proxy v0.0.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.2 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/rogpeppe/go-internal v1.16.0 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.28.0 // indirect
)

replace github.com/valguard-ai/valguard/apps/proxy => ../../apps/proxy
