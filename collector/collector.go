package collector

import (
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/leoluk/perflib_exporter/perflib"
	"github.com/prometheus/client_golang/prometheus"
)

// ...
const (
	Namespace = "perflib"

	// Conversion factors
	hundredNsToSecondsScaleFactor = 1 / 1e7
)

// Collector is the interface a collector has to implement.
type Collector interface {
	// Get new metrics and expose them via prometheus registry.
	Collect(ch chan<- prometheus.Metric) (err error)
}

type CounterKey struct {
	ObjectIndex  uint
	CounterIndex uint
	CounterType  uint32 // This is a bit mask
}

func NewCounterKey(object *perflib.PerfObject, def *perflib.PerfCounterDef) CounterKey {
	return CounterKey{object.NameIndex, def.NameIndex, def.CounterType}
}

type PerflibCollector struct {
	perflibQuery string
	perflibDescs map[CounterKey]*prometheus.Desc
	logger       log.Logger
}

func NewPerflibCollector(l log.Logger, query string) (c PerflibCollector) {
	c.perflibQuery = query
	c.logger = l

	objects, err := perflib.QueryPerformanceData(c.perflibQuery)

	if err != nil {
		panic(err)
	}

	level.Debug(c.logger).Log("object_count", len(objects))

	c.perflibDescs = make(map[CounterKey]*prometheus.Desc)

	for _, object := range objects {
		for _, def := range object.CounterDefs {
			desc := descFromCounterDef(*object, *def)

			key := NewCounterKey(object, def)
			c.perflibDescs[key] = desc
		}
	}

	return
}

func (c PerflibCollector) Collect(ch chan<- prometheus.Metric) (err error) {
	// TODO QueryPerformanceData timing metric
	objects, err := perflib.QueryPerformanceData(c.perflibQuery)

	if err != nil {
		// TODO - we shouldn't panic if a single call fails
		panic(err)
	}

	level.Debug(c.logger).Log("object_count", len(objects))

	for _, object := range objects {
		n := object.NameIndex

		for _, instance := range object.Instances {
			name := instance.Name

			// _Total metrics do not fit into the Prometheus model - we try to merge similar
			// metrics and give them labels, so you'd sum() them instead. Having a _Total label
			// would make
			if strings.HasSuffix(name, "_Total") || strings.HasPrefix(name, "Total") {
				continue
			}

			for _, counter := range instance.Counters {
				if IsDefPromotedLabel(n, counter.Def.NameIndex) {
					continue
				}

				if counter == nil {
					level.Debug(c.logger).Log("msg", "nil counter", "object", object.Name, "instance", instance.Name)
					continue
				}

				if counter.Def.NameIndex == 0 {
					level.Debug(c.logger).Log("msg", "null counter", "object", object.Name, "instance", instance.Name)
					continue
				}

				if counter.Def.Name == "" {
					level.Debug(c.logger).Log("msg", "no counter", "object", object.Name, "instance", instance.Name)
					continue
				}

				if counter.Def.Name == "No name" {
					level.Debug(c.logger).Log("msg", "no name counter", "object", object.Name, "instance", instance.Name, "counter", counter.Def.Name)
					continue
				}

				key := NewCounterKey(object, counter.Def)

				desc, ok := c.perflibDescs[key]

				if !ok {
					level.Debug(c.logger).Log("msg", "missing metric description for counter", "object", object.Name, "instance", instance.Name, "counter", counter.Def.Name)
					continue
				}

				labels := []string{name}

				if len(object.Instances) == 1 {
					labels = []string{}
				}

				if HasPromotedLabels(n) {
					labels = append(labels, PromotedLabelValuesForInstance(n, instance)...)
				}

				// TODO - Label merging needs to be fixed for [230] Process
				//if HasMergedLabels(n) {
				//	_, value := MergedMetricForInstance(n, counter.Def.NameIndex)
				//
				//	// Null string in definition means we should skip this metric (it's probably a sum)
				//	if value == "" {
				//		log.Debugf("Skipping %d -> %s (empty merge label)", n, counter.Def.NameIndex)
				//		continue
				//	}
				//	labels = append(labels, value)
				//}

				valueType, err := GetPrometheusValueType(counter.Def.CounterType)

				if err != nil {
					// TODO - Is this too verbose? There will always be counter types we don't support
					level.Debug(c.logger).Log("err", err)
					continue
				}

				value := float64(counter.Value)

				if counter.Def.IsNanosecondCounter {
					value = value * hundredNsToSecondsScaleFactor
				}

				if IsElapsedTime(counter.Def.CounterType) {
					// convert from Windows timestamp (1 jan 1601) to unix timestamp (1 jan 1970)
					value = float64(counter.Value-116444736000000000) / float64(object.Frequency)
				}

				metric := prometheus.MustNewConstMetric(
					desc,
					valueType,
					value,
					labels...,
				)

				ch <- metric
			}
		}
	}

	return nil
}
