package collector

import (
	"fmt"

	"github.com/leoluk/perflib_exporter/perflib"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
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

type PerflibCollector struct {
	perflibQuery   string
	perflibObjects []*perflib.PerfObject
	perflibDescs   map[uint]*prometheus.Desc
}

var countersPerDef map[uint]uint

func NewPerflibCollector(query string) (c PerflibCollector) {
	c.perflibQuery = query

	objects, err := perflib.QueryPerformanceData(c.perflibQuery)

	if err != nil {
		panic(err)
	}

	c.perflibObjects = objects
	log.Debugf("Number of objects: %d", len(objects))

	c.perflibDescs = make(map[uint]*prometheus.Desc)

	knownNames := make(map[string]bool)

	for _, object := range objects {
		for _, def := range object.CounterDefs {
			name, desc := descFromCounterDef(*object, *def)
			keyname := fmt.Sprintf("%s|%s", object.Name, name)
			if _, ok := knownNames[keyname]; ok {
				continue
			}

			c.perflibDescs[def.NameIndex] = desc
			knownNames[keyname] = true
		}
	}

	// TODO: we do not handle multi-value counters yet, so we count and remove them
	countersPerDef = make(map[uint]uint)

	for _, object := range objects {
		instance := object.Instances[0]
		for _, counter := range instance.Counters {
			countersPerDef[counter.Def.NameIndex] += 1
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

	log.Debugf("Number of objects: %d", len(objects))

	for _, object := range objects {
		n := object.NameIndex

		for _, instance := range object.Instances {
			name := instance.Name

			// _Total metrics do not fit into the Prometheus model - we try to merge similar
			// metrics and give them labels, so you'd sum() them instead. Having a _Total label
			// would make

			for _, counter := range instance.Counters {
				if IsDefPromotedLabel(n, counter.Def.NameIndex) {
					continue
				}

				if counter == nil {
					log.Debugf("nil counter for %s -> %s", object.Name, instance.Name)
					continue
				}

				if counter.Def.NameIndex == 0 {
					log.Debugf("null counter index for %s -> %s", object.Name, instance.Name)
					continue
				}

				if countersPerDef[counter.Def.NameIndex] > 1 {
					log.Debugf("multi counter %s -> %s -> %s", object.Name, instance.Name, counter.Def.Name)
					continue
				}

				if counter.Def.Name == "No name" {
					log.Debugf("no name counter %s -> %s -> %s", object.Name, instance.Name, counter.Def.Name)
					continue
				}

				desc, ok := c.perflibDescs[counter.Def.NameIndex]

				if !ok {
					log.Debugf("missing metric description for counter %s -> %s -> %s", object.Name, instance.Name, counter.Def.Name)
					continue
				}

				labels := []string{name}

				if counter.Def.Name == "" {
					labels = []string{}
				}

				if HasPromotedLabels(n) {
					labels = append(labels, PromotedLabelValuesForInstance(n, instance)...)
				}

				if HasMergedLabels(n) {
					_, value := MergedMetricForInstance(n, counter.Def.NameIndex)

					// Null string in definition means we should skip this metric (it's probably a sum)
					if value == "" {
						log.Debugf("Skipping %d -> %s (empty merge label)", n, counter.Def.NameIndex)
						continue
					}
					labels = append(labels, value)
				}

				value := float64(counter.Value)

				if counter.Def.IsNanosecondCounter {
					value = value * hundredNsToSecondsScaleFactor
				}

				metric := prometheus.MustNewConstMetric(
					desc,
					prometheus.CounterValue,
					value,
					labels...,
				)

				ch <- metric
			}
		}
	}

	/*ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(),
		prometheus.CounterValue,
		float64(0),
		"ds_client",
	)*/

	return nil
}
