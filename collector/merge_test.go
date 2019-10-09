// +build windows

package collector

import "fmt"

func ExampleMergedLabelsForInstance() {
	fmt.Println(MergedLabelsForInstance(230, 142))

	// Output:
	// processor_time_total mode
}

func ExampleMergedMetricForInstance() {
	fmt.Println(MergedMetricForInstance(230, 142))

	// Output:
	// processor_time_total user
}
