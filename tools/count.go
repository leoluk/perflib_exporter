package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/leoluk/perflib_exporter/perflib"
)

func main() {
	if len(os.Args) <= 1 {
		log.Fatalln("Usage: ./count.exe Global")
	}

	numCounters := 0

	query := strings.Join(os.Args[1:], " ")

	objects, err := perflib.QueryPerformanceData(query)

	if err != nil {
		panic(err)
	}

	for _, object := range objects {
		for _, instance := range object.Instances {
			for _ = range instance.Counters {
				numCounters += 1
			}
		}
	}

	fmt.Printf("\nNumber of counters: %d\n", numCounters)
}
