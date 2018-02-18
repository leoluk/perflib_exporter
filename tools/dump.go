package main

import (
	"fmt"
	"strings"
	"os"
	"log"

	"github.com/leoluk/perflib_exporter/perflib"
)

func main() {
	if len(os.Args) <= 1 {
		log.Fatalln("Usage: ./dump.exe Global")
	}

	numCounters := 0

	query := strings.Join(os.Args[1:], " ")

	objects, err := perflib.QueryPerformanceData(query)

	if err != nil {
		panic(err)
	}

	for _, object := range objects {
		fmt.Printf("%d %s [%d counters, %d instance(s)]\n",
			object.NameIndex, object.Name, len(object.CounterDefs), len(object.Instances))

		for _, instance := range object.Instances {
			if len(instance.Name) > 0 {
				fmt.Printf("`-- \"%s\"\n", instance.Name)
			} else {
				fmt.Println("`-- (default)")
			}

			for _, counter := range instance.Counters {
				fmt.Printf("    `-- %s [%d] = %d\n", counter.Def.Name, counter.Def.NameIndex, counter.Value)
				numCounters += 1
			}
		}
	}

	fmt.Printf("\nNumber of objects: %d\n", len(objects))
	fmt.Printf("\nNumber of counters: %d\n", numCounters)
}
