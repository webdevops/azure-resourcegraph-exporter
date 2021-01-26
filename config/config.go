package config

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type (
	Config struct {
		Queries []ConfigQuery `yaml:"queries"`
	}

	ConfigQuery struct {
		Module        string    `yaml:"module"`
		Metric        string    `yaml:"metric"`
		Query         string    `yaml:"query"`
		Subscriptions *[]string `yaml:"subscriptions"`
		ValueColumn   string    `yaml:"valueColumn"`
	}
)

func NewConfig(path string) (config Config) {
	var filecontent []byte

	config = Config{}

	log.Infof("reading configuration from file %v", path)
	if data, err := ioutil.ReadFile(path); err == nil {
		filecontent = data
	} else {
		panic(err)
	}

	log.Info("parsing configuration")
	if err := yaml.Unmarshal(filecontent, &config); err != nil {
		panic(err)
	}

	return
}
