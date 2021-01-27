package main

import "github.com/prometheus/client_golang/prometheus"

type (
	MetricList struct {
		list map[string][]MetricRow
	}

	MetricRow struct {
		labels prometheus.Labels
		value  float64
	}
)

func (l *MetricList) Init() {
	l.list = map[string][]MetricRow{}
}

func (l *MetricList) Add(name string, metric ...MetricRow) {
	if _, ok := l.list[name]; !ok {
		l.list[name] = []MetricRow{}
	}

	l.list[name] = append(l.list[name], metric...)
}

func (l *MetricList) GetMetricNames() []string {
	list := []string{}

	for name := range l.list {
		list = append(list, name)
	}

	return list
}

func (l *MetricList) GetMetricList(name string) []MetricRow {
	return l.list[name]
}

func (l *MetricList) GetMetricLabelNames(name string) []string {
	uniqueLabelMap := map[string]string{}

	for _, row := range l.list[name] {
		for labelName := range row.labels {
			uniqueLabelMap[labelName] = labelName
		}
	}

	list := []string{}
	for labelName := range uniqueLabelMap {
		list = append(list, labelName)
	}

	return list
}
