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
		Module        string              `yaml:"module"`
		Metric        string              `yaml:"metric"`
		Query         string              `yaml:"query"`
		Subscriptions *[]string           `yaml:"subscriptions"`
		IdField       string              `yaml:"idField"`
		ValueField    string              `yaml:"valueField"`
		AutoExpand    SingleOrMultiString `yaml:"autoExpand"`
	}

	SingleOrMultiString struct {
		Values []string
	}
)

func (sm *SingleOrMultiString) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var multi []string
	err := unmarshal(&multi)
	if err != nil {
		var single string
		err := unmarshal(&single)
		if err != nil {
			return err
		}
		sm.Values = make([]string, 1)
		sm.Values[0] = single
	} else {
		sm.Values = multi
	}
	return nil
}

func (c *ConfigQuery) IsAutoExpandColumn(propertyName string) bool {
	for _, name := range c.AutoExpand.Values {
		if name == "*" || name == propertyName {
			return true
		}
	}

	return false
}

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
