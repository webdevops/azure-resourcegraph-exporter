package config

import (
	"encoding/json"
	"time"
)

type (
	Opts struct {
		// logger
		Logger struct {
			Level  string `long:"log.level"    env:"LOG_LEVEL"   description:"Log level" choice:"trace" choice:"debug" choice:"info" choice:"warning" choice:"error" default:"info"`                          // nolint:staticcheck // multiple choices are ok
			Format string `long:"log.format"   env:"LOG_FORMAT"  description:"Log format" choice:"logfmt" choice:"json" default:"logfmt"`                                                                     // nolint:staticcheck // multiple choices are ok
			Source string `long:"log.source"   env:"LOG_SOURCE"  description:"Show source for every log message (useful for debugging and bug reports)" choice:"" choice:"short" choice:"file" choice:"full"` // nolint:staticcheck // multiple choices are ok
			Color  string `long:"log.color"    env:"LOG_COLOR"   description:"Enable color for logs" choice:"" choice:"auto" choice:"yes" choice:"no"`                                                        // nolint:staticcheck // multiple choices are ok
			Time   bool   `long:"log.time"     env:"LOG_TIME"    description:"Show log time"`
		}

		// azure
		Azure struct {
			Environment  *string  `long:"azure-environment"            env:"AZURE_ENVIRONMENT"                description:"Azure environment name" default:"AZUREPUBLICCLOUD"`
			Subscription []string `long:"azure-subscription"           env:"AZURE_SUBSCRIPTION_ID"     env-delim:" "  description:"Azure subscription ID"`
		}

		// config
		Config struct {
			Path string `long:"config" short:"c"  env:"CONFIG"   description:"Config path" required:"true"`
		}

		// general options
		Server struct {
			// general options
			Bind         string        `long:"server.bind"              env:"SERVER_BIND"           description:"Server address"        default:":8080"`
			ReadTimeout  time.Duration `long:"server.timeout.read"      env:"SERVER_TIMEOUT_READ"   description:"Server read timeout"   default:"5s"`
			WriteTimeout time.Duration `long:"server.timeout.write"     env:"SERVER_TIMEOUT_WRITE"  description:"Server write timeout"  default:"10s"`
		}
	}
)

func (o *Opts) GetJson() []byte {
	jsonBytes, err := json.Marshal(o)
	if err != nil {
		panic(err)
	}
	return jsonBytes
}
