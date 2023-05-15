package internal

import "net"

// Config is the server configuration.
type Config struct {
	Address          string     `env:"SERVER_ADDRESS" json:"server_address"`
	GRPCAddress      string     `env:"GRPC_SERVER_ADDRESS" json:"grpc_server_address"`
	BaseURL          string     `env:"BASE_URL" json:"base_url"`
	FileStoragePath  string     `env:"FILE_STORAGE_PATH" json:"file_storage_path"`
	DatabaseDSN      string     `env:"DATABASE_DSN" json:"database_dsn"`
	EnableHTTPS      bool       `env:"ENABLE_HTTPS" json:"enable_https"`
	GRPCEnableTLS    bool       `env:"GRPC_ENABLE_TLS" json:"grpc_enable_tls"`
	TrustedSubnetStr string     `env:"TRUSTED_SUBNET" json:"trusted_subnet"`
	TrustedSubnet    *net.IPNet `json:"_"`
}
