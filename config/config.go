package config

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
)

type (
	Config struct {
		Queries []ConfigQuery `yaml:"queries"`
	}

	ConfigQuery struct {
		MetricConfig  ConfigQueryMetric `yaml:",inline"`
		Module        string            `yaml:"module"`
		Query         string            `yaml:"query"`
		Subscriptions *[]string         `yaml:"subscriptions"`
	}

	ConfigQueryMetric struct {
		Metric     string                   `yaml:"metric"`
		Value      *float64                 `yaml:"value"`
		AutoExpand bool                     `yaml:"autoExpand"`
		Fields     []ConfigQueryMetricField `yaml:"fields"`
	}

	ConfigQueryMetricField struct {
		Name    string             `yaml:"name"`
		Target  string             `yaml:"target"`
		Type    string             `yaml:"type"`
		Filters []string           `yaml:"filters"`
		Expand  *ConfigQueryMetric `yaml:"metric"`
	}

	SingleOrMultiString struct {
		Values []string
	}
)

func (m *ConfigQueryMetric) IsExpand(field string) bool {
	if m.AutoExpand {
		return true
	}

	for _, fieldConfig := range m.Fields {
		if fieldConfig.Name == field {
			if fieldConfig.Type == "expand" || fieldConfig.Expand != nil {
				return true
			}
			break
		}
	}

	return false
}

func (m *ConfigQueryMetric) GetFieldConfigMap() (list map[string]ConfigQueryMetricField) {
	list = map[string]ConfigQueryMetricField{}

	for _, field := range m.Fields {
		list[field.Name] = field
	}

	return
}

func (f *ConfigQueryMetricField) IsIgnore() bool {
	return f.Type == "ignore"
}

func (f *ConfigQueryMetricField) IsId() bool {
	return f.Type == "id"
}

func (f *ConfigQueryMetricField) IsValue() bool {
	return f.Type == "value"
}

func (f *ConfigQueryMetricField) GetTargetFieldName(sourceName string) (ret string) {
	ret = sourceName
	if f.Target != "" {
		ret = f.Target
	} else if f.Name != "" {
		ret = f.Name
	}
	return
}

func (f *ConfigQueryMetricField) TransformString(value string) (ret string) {
	ret = value

	for _, filter := range f.Filters {
		switch strings.ToLower(filter) {
		case "tolower":
			ret = strings.ToLower(ret)
		case "toupper":
			ret = strings.ToUpper(ret)
		case "totitle":
			ret = strings.ToTitle(ret)
		}
	}
	return
}

func (f *ConfigQueryMetricField) TransformFloat64(value float64) (ret string) {
	ret = fmt.Sprintf("%v", value)
	ret = f.TransformString(ret)
	return
}

func (f *ConfigQueryMetricField) TransformBool(value bool) (ret string) {
	if value {
		ret = "true"
	} else {
		ret = "false"
	}
	ret = f.TransformString(ret)
	return
}

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
