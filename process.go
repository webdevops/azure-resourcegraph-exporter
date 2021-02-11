package main

import (
	"github.com/webdevops/azure-resourcegraph-exporter/config"
	"strconv"
)

func convertSubMetricInterfaceToArray(val interface{}) []interface{} {
	// convert to array even if not array
	ret := []interface{}{}
	switch v := val.(type) {
	case map[string]interface{}:
		ret = append(ret, v)
	case []interface{}:
		ret = v
	}

	return ret
}

func processFieldAndAddToMetric(fieldName string, value interface{}, fieldConfig config.ConfigQueryMetricField, metric *MetricRow) {
	labelName := fieldConfig.GetTargetFieldName(fieldName)

	switch v := value.(type) {
	case string:
		v = fieldConfig.TransformString(v)
		if fieldConfig.IsTypeValue() {
			if value, err := strconv.ParseFloat(v, 64); err == nil {
				metric.Value = value
			} else {
				metric.Value = 0
			}
		} else {
			metric.Labels[labelName] = v
		}
	case int64:
		fieldValue := fieldConfig.TransformFloat64(float64(v))

		if fieldConfig.IsTypeValue() {
			metric.Value = float64(v)
		} else {
			metric.Labels[labelName] = fieldValue
		}
	case float64:
		fieldValue := fieldConfig.TransformFloat64(v)

		if fieldConfig.IsTypeValue() {
			metric.Value = v
		} else {
			metric.Labels[labelName] = fieldValue
		}
	case bool:
		fieldValue := fieldConfig.TransformBool(v)
		if fieldConfig.IsTypeValue() {
			if v {
				metric.Value = 1
			} else {
				metric.Value = 0
			}
		} else {
			metric.Labels[labelName] = fieldValue
		}
	}
}
