package config

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"regexp"
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
		Name    string                         `yaml:"name"`
		Target  string                         `yaml:"target"`
		Type    string                         `yaml:"type"`
		Filters []ConfigQueryMetricFieldFilter `yaml:"filters"`
		Expand  *ConfigQueryMetric             `yaml:"metric"`
	}

	ConfigQueryMetricFieldFilter struct {
		Type         string `yaml:"type"`
		RegExp       string `yaml:"regexp"`
		Replacement  string `yaml:"replacement"`
		parsedRegexp *regexp.Regexp
	}

	ConfigQueryMetricFieldFilterParser struct {
		Type        string `yaml:"type"`
		RegExp      string `yaml:"regexp"`
		Replacement string `yaml:"replacement"`
	}
)

func (c *Config) Validate() error {
	if len(c.Queries) == 0 {
		return errors.New("no queries found")
	}

	for _, queryConfig := range c.Queries {
		if err := queryConfig.Validate(); err != nil {
			return fmt.Errorf("query \"%v\": %v", queryConfig.MetricConfig.Metric, err)
		}
	}

	return nil
}

func (c *ConfigQuery) Validate() error {
	if err := c.MetricConfig.Validate(); err != nil {
		return err
	}

	return nil
}

func (c *ConfigQueryMetric) Validate() error {
	if c.Metric == "" {
		return errors.New("no metric name set")
	}

	for _, field := range c.Fields {
		if err := field.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (c *ConfigQueryMetricField) Validate() error {
	if c.Name == "" {
		return errors.New("no field name set")
	}

	for _, filter := range c.Filters {
		if err := filter.Validate(); err != nil {
			return fmt.Errorf("field \"%v\": %v", c.Name, err)
		}
	}

	return nil
}

func (c *ConfigQueryMetricFieldFilter) Validate() error {
	if c.Type == "" {
		return errors.New("no type name set")
	}

	switch strings.ToLower(c.Type) {
	case "tolower":
	case "toupper":
	case "totitle":
	case "regexp":
		if c.RegExp == "" {
			return errors.New("no regexp for filter set")
		}
		c.parsedRegexp = regexp.MustCompile(c.RegExp)
	default:
		return fmt.Errorf("filter \"%v\" not supported", c.Type)
	}

	return nil
}

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
		switch strings.ToLower(filter.Type) {
		case "tolower":
			ret = strings.ToLower(ret)
		case "toupper":
			ret = strings.ToUpper(ret)
		case "totitle":
			ret = strings.ToTitle(ret)
		case "regexp":
			if filter.parsedRegexp == nil {
				filter.parsedRegexp = regexp.MustCompile(filter.RegExp)
			}
			ret = filter.parsedRegexp.ReplaceAllString(value, filter.Replacement)
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

func (f *ConfigQueryMetricFieldFilter) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var multi ConfigQueryMetricFieldFilterParser
	err := unmarshal(&multi)
	if err != nil {
		var single string
		err := unmarshal(&single)
		if err != nil {
			return err
		}
		f.Type = single
	} else {
		f.Type = multi.Type
		f.RegExp = multi.RegExp
		f.Replacement = multi.Replacement
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
