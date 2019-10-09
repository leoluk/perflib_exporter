// +build windows

package collector

import (
	"fmt"

	"github.com/leoluk/perflib_exporter/perflib"
)

func ExampleLabelsForObject() {
	fmt.Println(PromotedLabelsForObject(230))

	// Output:
	// [process_id creating_process_id]
}

func ExamplePromotedLabelValuesForInstance() {
	// Process
	objects, err := perflib.QueryPerformanceData("230")

	if err != nil {
		panic(err)
	}

	// First instance is "Idle"
	instance := objects[0].Instances[0]
	fmt.Println(instance.Name)

	values := PromotedLabelValuesForInstance(230, instance)

	fmt.Println(values)

	// Output:
	// Idle
	// [0 0]
}
