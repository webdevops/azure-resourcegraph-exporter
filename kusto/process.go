package kusto

import (
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

func processFieldAndAddToMetric(fieldName string, value interface{}, fieldConfig ConfigQueryMetricField, metric *MetricRow) {
	labelName := fieldConfig.GetTargetFieldName(fieldName)

	// upgrade to 64bit
	switch v := value.(type) {
	// ----------------------------------------------------
	// int
	case int8:
		value = int64(v)
	case uint8:
		value = int64(v)
	case int16:
		value = int64(v)
	case uint16:
		value = int64(v)
	case int32:
		value = int64(v)
	case uint32:
		value = int64(v)
	// ----------------------------------------------------
	// float
	case float32:
		value = float64(v)
	}

	switch v := value.(type) {
	// ----------------------------------------------------
	// string
	case string:
		v = fieldConfig.TransformString(v)
		if fieldConfig.IsTypeValue() {
			if value, err := strconv.ParseFloat(v, 64); err == nil {
				metric.Value = &value
			} else {
				metric.Value = nil
			}
		} else {
			metric.Labels[labelName] = v
		}
	// ----------------------------------------------------
	// int
	case uint64:
		fieldValue := fieldConfig.TransformFloat64(float64(v))
		if fieldConfig.IsTypeValue() {
			metric.Value = toFloat64Ptr(float64(v))
		} else {
			metric.Labels[labelName] = fieldValue
		}
	case int64:
		fieldValue := fieldConfig.TransformFloat64(float64(v))
		if fieldConfig.IsTypeValue() {
			metric.Value = toFloat64Ptr(float64(v))
		} else {
			metric.Labels[labelName] = fieldValue
		}

	// ----------------------------------------------------
	// float
	case float64:
		fieldValue := fieldConfig.TransformFloat64(v)
		if fieldConfig.IsTypeValue() {
			metric.Value = toFloat64Ptr(v)
		} else {
			metric.Labels[labelName] = fieldValue
		}

	// ----------------------------------------------------
	// boolean
	case bool:
		fieldValue := fieldConfig.TransformBool(v)
		if fieldConfig.IsTypeValue() {
			if v {
				metric.Value = toFloat64Ptr(1)
			} else {
				metric.Value = toFloat64Ptr(0)
			}
		} else {
			metric.Labels[labelName] = fieldValue
		}

	// ----------------------------------------------------
	// nil
	case nil:
		if fieldConfig.IsTypeValue() {
			metric.Value = nil
		} else {
			metric.Labels[labelName] = ""
		}
	}
}
