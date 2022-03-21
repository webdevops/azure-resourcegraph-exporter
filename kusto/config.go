package kusto

import (
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const (
	MetricFieldTypeExpand  = "expand"
	MetricFieldTypeIgnore  = "ignore"
	MetricFieldTypeId      = "id"
	MetricFieldTypeValue   = "value"
	MetricFieldTypeDefault = "string"
	MetricFieldTypeBool    = "bool"
	MetricFieldTypeBoolean = "boolean"

	MetricFieldFilterToLower    = "tolower"
	MetricFieldFilterToUpper    = "toupper"
	MetricFieldFilterToTitle    = "totitle"
	MetricFieldFilterToRegexp   = "regexp"
	MetricFieldFilterToUnixtime = "tounixtime"
)

type (
	Config struct {
		Queries []ConfigQuery `yaml:"queries"`
	}

	ConfigQuery struct {
		MetricConfig  ConfigQueryMetric `yaml:",inline"`
		QueryMode     string            `yaml:"queryMode"`
		Metric        string            `yaml:"metric"`
		Module        string            `yaml:"module"`
		Query         string            `yaml:"query"`
		Timespan      *string           `yaml:"timespan"`
		Subscriptions *[]string         `yaml:"subscriptions"`
	}

	ConfigQueryMetric struct {
		Value        *float64                 `yaml:"value"`
		Fields       []ConfigQueryMetricField `yaml:"fields"`
		Labels       map[string]string        `yaml:"labels"`
		DefaultField ConfigQueryMetricField   `yaml:"defaultField"`
		Publish      *bool                    `yaml:"publish"`
	}

	ConfigQueryMetricField struct {
		Name    string                         `yaml:"name"`
		Metric  string                         `yaml:"metric"`
		Source  string                         `yaml:"source"`
		Target  string                         `yaml:"target"`
		Type    string                         `yaml:"type"`
		Labels  map[string]string              `yaml:"labels"`
		Filters []ConfigQueryMetricFieldFilter `yaml:"filters"`
		Expand  *ConfigQueryMetric             `yaml:"expand"`
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
			return fmt.Errorf("query \"%v\": %v", queryConfig.Metric, err)
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
	// validate default field
	c.DefaultField.Name = "default"
	if err := c.DefaultField.Validate(); err != nil {
		return err
	}

	// validate fields
	for _, field := range c.Fields {
		if err := field.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (c *ConfigQueryMetric) IsPublished() bool {
	if c.Publish != nil {
		return *c.Publish
	}

	return true
}

func (c *ConfigQueryMetricField) Validate() error {
	if c.Name == "" {
		return errors.New("no field name set")
	}

	switch c.GetType() {
	case MetricFieldTypeDefault:
	case MetricFieldTypeBool:
	case MetricFieldTypeBoolean:
	case MetricFieldTypeExpand:
	case MetricFieldTypeId:
	case MetricFieldTypeValue:
	case MetricFieldTypeIgnore:
	default:
		return fmt.Errorf("field \"%s\": unsupported type \"%s\"", c.Name, c.GetType())
	}

	for _, filter := range c.Filters {
		if err := filter.Validate(); err != nil {
			return fmt.Errorf("field \"%v\": %v", c.Name, err)
		}
	}

	return nil
}

func (c *ConfigQueryMetricField) GetType() (ret string) {
	ret = strings.ToLower(c.Type)

	if ret == "" {
		ret = MetricFieldTypeDefault
	}

	return
}

func (c *ConfigQueryMetricFieldFilter) Validate() error {
	if c.Type == "" {
		return errors.New("no type name set")
	}

	switch strings.ToLower(c.Type) {
	case MetricFieldFilterToLower:
	case MetricFieldFilterToUpper:
	case MetricFieldFilterToTitle:
	case MetricFieldFilterToRegexp:
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
	for _, fieldConfig := range m.Fields {
		if fieldConfig.Name == field {
			if fieldConfig.IsExpand() {
				return true
			}
			break
		}
	}

	return false
}

func (m *ConfigQueryMetric) GetFieldConfigMap() (list map[string][]ConfigQueryMetricField) {
	list = map[string][]ConfigQueryMetricField{}

	for _, field := range m.Fields {
		if _, ok := list[field.Name]; !ok {
			list[field.Name] = []ConfigQueryMetricField{}
		}
		list[field.GetSourceField()] = append(list[field.GetSourceField()], field)
	}

	return
}

func (f *ConfigQueryMetricField) GetSourceField() (ret string) {
	ret = f.Source
	if ret == "" {
		ret = f.Name
	}
	return
}

func (f *ConfigQueryMetricField) IsExpand() bool {
	return f.Type == MetricFieldTypeExpand || f.Expand != nil
}

func (f *ConfigQueryMetricField) IsSourceField() bool {
	return f.Source != ""
}

func (f *ConfigQueryMetricField) IsTypeIgnore() bool {
	return f.GetType() == MetricFieldTypeIgnore
}

func (f *ConfigQueryMetricField) IsTypeId() bool {
	return f.GetType() == MetricFieldTypeId
}

func (f *ConfigQueryMetricField) IsTypeValue() bool {
	return f.GetType() == MetricFieldTypeValue
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

	switch f.Type {
	case MetricFieldTypeBoolean:
		fallthrough
	case MetricFieldTypeBool:
		switch strings.ToLower(ret) {
		case "1":
			fallthrough
		case "true":
			fallthrough
		case "yes":
			ret = "true"
		default:
			ret = "false"
		}
	}
	for _, filter := range f.Filters {
		switch strings.ToLower(filter.Type) {
		case MetricFieldFilterToLower:
			ret = strings.ToLower(ret)
		case MetricFieldFilterToUpper:
			ret = strings.ToUpper(ret)
		case MetricFieldFilterToTitle:
			ret = strings.ToTitle(ret)
		case MetricFieldFilterToUnixtime:
			ret = convertStringToUnixtime(ret)
		case MetricFieldFilterToRegexp:
			if filter.parsedRegexp == nil {
				filter.parsedRegexp = regexp.MustCompile(filter.RegExp)
			}
			ret = filter.parsedRegexp.ReplaceAllString(ret, filter.Replacement)
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
	/*  #nosec G304 */
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
