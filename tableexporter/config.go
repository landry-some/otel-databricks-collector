package tableexporter

import "go.opentelemetry.io/collector/component"

type Config struct {
	component.Config

	DBHost      string `mapstructure:"db_host"`
	DBHttpPath  string `mapstructure:"db_http_path"`
	DBToken     string `mapstructure:"db_token"`
	TargetTable string `mapstructure:"target_table"`
}
