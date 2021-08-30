package kusto

import (
	"fmt"
)

func BuildPrometheusMetricList(name string, metricConfig ConfigQueryMetric, row map[string]interface{}) (list map[string][]MetricRow) {
	list = map[string][]MetricRow{}
	idFieldList := map[string]string{}

	mainMetrics := map[string]*MetricRow{}
	mainMetrics[name] = NewMetricRow()

	fieldConfigMap := metricConfig.GetFieldConfigMap()

	// add default value to main metric (if set)
	if metricConfig.Value != nil {
		mainMetrics[name].Value = metricConfig.Value
	}

	// additional labels
	for rowName, rowValue := range metricConfig.Labels {
		mainMetrics[name].Labels[rowName] = rowValue
	}

	// main metric
	for fieldName, rowValue := range row {
		if fieldConfList, ok := fieldConfigMap[fieldName]; ok {
			// field configuration available
			for _, fieldConfig := range fieldConfList {
				if fieldConfig.IsTypeIgnore() {
					continue
				}

				if fieldConfig.IsExpand() {
					continue
				}

				var metricRow *MetricRow
				isNewMetricRow := false

				if fieldConfig.Metric != "" {
					// field has own metric name, assuming individual metric row
					isNewMetricRow = true
					metricRow = NewMetricRow()
				} else {
					// field hasn't an own metric name, merge with main metric
					fieldConfig.Metric = name
					metricRow = mainMetrics[fieldConfig.Metric]
				}

				processFieldAndAddToMetric(fieldName, rowValue, fieldConfig, metricRow)

				// additional labels
				for rowName, rowValue := range fieldConfig.Labels {
					metricRow.Labels[rowName] = rowValue
				}

				// id labels
				if fieldConfig.IsTypeId() {
					labelName := fieldConfig.GetTargetFieldName(fieldName)
					if _, ok := mainMetrics[name].Labels[labelName]; ok {
						idFieldList[labelName] = metricRow.Labels[labelName]
					}
				}

				// check if metric should be skipped
				if isNewMetricRow {
					// save as own metric
					if _, ok := list[fieldConfig.Metric]; !ok {
						list[fieldConfig.Metric] = []MetricRow{}
					}
					list[fieldConfig.Metric] = append(list[fieldConfig.Metric], *metricRow)
				} else {
					// set to main metric row
					mainMetrics[fieldConfig.Metric] = metricRow
				}
			}
		} else {
			// no field config, fall back to "defaultField"
			fieldConfig := metricConfig.DefaultField
			if !fieldConfig.IsTypeIgnore() {
				processFieldAndAddToMetric(fieldName, rowValue, fieldConfig, mainMetrics[name])
			}
		}
	}

	// sub metrics (aka nested/expand structures)
	for fieldName, rowValue := range row {
		if !metricConfig.IsExpand(fieldName) {
			continue
		}

		for _, rowValue := range convertSubMetricInterfaceToArray(rowValue) {
			if v, ok := rowValue.(map[string]interface{}); ok {
				if fieldConfList, ok := fieldConfigMap[fieldName]; ok {
					for _, fieldConfig := range fieldConfList {
						if fieldConfig.IsTypeIgnore() {
							continue
						}

						// add fieldname to metric if no custom metric is set
						if fieldConfig.Metric == "" {
							fieldConfig.Metric = fmt.Sprintf("%s_%s", name, fieldName)
						}

						subMetricConfig := ConfigQueryMetric{}
						if fieldConfig.Expand != nil {
							subMetricConfig = *fieldConfig.Expand
						}

						subMetricList := BuildPrometheusMetricList(fieldConfig.Metric, subMetricConfig, v)

						for subMetricName, subMetricList := range subMetricList {
							if _, ok := list[subMetricName]; !ok {
								list[subMetricName] = []MetricRow{}
							}
							list[subMetricName] = append(list[subMetricName], subMetricList...)
						}
					}
				}
			}
		}
	}

	// add main metrics
	if metricConfig.IsPublished() {
		for metricName, metricRow := range mainMetrics {
			if _, ok := list[metricName]; !ok {
				list[metricName] = []MetricRow{}
			}
			list[metricName] = append(list[metricName], *metricRow)
		}
	}

	// add id labels
	for metricName := range list {
		for row := range list[metricName] {
			for idLabel, idValue := range idFieldList {
				list[metricName][row].Labels[idLabel] = idValue
			}
		}
	}

	return
}
