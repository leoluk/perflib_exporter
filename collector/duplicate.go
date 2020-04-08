package collector

import (
	"github.com/leoluk/perflib_exporter/perflib"
	"strconv"
)

var duplicates = map[*perflib.PerfObject]bool{}

// RequiresDuplicateDistinction determines whether the Perflib object contains instances with duplicate names.
func RequiresDuplicateDistinction(object *perflib.PerfObject) bool {
	return indexDuplicateDistinctionPool(object)
}

// DuplicateDistinctionLabel returns the labels to allow duplicate distinction.
func DuplicateDistinctionLabel() []string {
	return []string{"instance_unique_id", "instance_index"}
}

// DuplicateDistinctionLabelValue returns the label values to allow duplicate distinction.
func DuplicateDistinctionLabelValue(instance *perflib.PerfInstance, index int) []string {
	return []string{strconv.Itoa(int(instance.UniqueID())), strconv.Itoa(index)}
}

// indexDuplicateDistinctionPool indexes the Perflib object instances for duplicate distinction.
func indexDuplicateDistinctionPool(object *perflib.PerfObject) bool {
	if _, ok := duplicates[object]; !ok {
		duplicates[object] = containsDuplicateNames(object.Instances)
	}
	return duplicates[object]
}

func containsDuplicateNames(instances []*perflib.PerfInstance) bool {
	var names = map[string]struct{}{}
	for _, instance := range instances {
		if _, duplicate := names[instance.Name]; duplicate {
			return true
		}
		names[instance.Name] = struct{}{}
	}
	return false
}
