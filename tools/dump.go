package main

import (
	"fmt"
	"strings"

	"github.com/alecthomas/kingpin"
	"github.com/davecgh/go-spew/spew"
	"github.com/leoluk/perflib_exporter/collector"
	"github.com/leoluk/perflib_exporter/perflib"
)

func main() {
	spew.Dump()
	var (
		args       = kingpin.Arg("query",
			"Perflib query").Required().Strings()
		showValues = kingpin.Flag("values",
			"Show counter values").Short('v').Bool()
		defsOnly   = kingpin.Flag("defs-only",
			"Show definitions only (no instances) and include Prometheus names").Short('o').Bool()
		unsorted   = kingpin.Flag("unsorted",
			"Do not sort objects").Short('u').Bool()
	)

	kingpin.Parse()

	query := strings.Join(*args, " ")

	objects, err := perflib.QueryPerformanceData(query)

	if !*unsorted {
		perflib.SortObjects(objects)
	}

	if err != nil {
		panic(err)
	}

	numCounters := 0
	numDefs := 0

	for _, o := range objects {
		if *defsOnly {
			fmt.Printf("%d %s [%d counters]\n",
				o.NameIndex, o.Name, len(o.CounterDefs))
		} else {
			fmt.Printf("%d %s [%d counters, %d instance(s)]\n",
				o.NameIndex, o.Name, len(o.CounterDefs), len(o.Instances))

		}

		for _, def := range o.CounterDefs {
			numDefs += 1
			if *defsOnly {
				fmt.Printf("    `-- [%d] %s \n", def.NameIndex, def.Name)
				fmt.Printf("        %s\n", collector.MakePrometheusLabel(def))
			}
		}

		if !*defsOnly {
			for _, instance := range o.Instances {
				if len(instance.Name) > 0 {
					fmt.Printf("`-- \"%s\"\n", instance.Name)
				} else {
					fmt.Println("`-- (default)")
				}

				for _, counter := range instance.Counters {
					if *showValues {
						fmt.Printf("    `-- %s [%d] = %d\n", counter.Def.Name, counter.Def.NameIndex, counter.Value)
					} else {
						fmt.Printf("    `-- %s [%d]\n", counter.Def.Name, counter.Def.NameIndex)
					}
					numCounters += 1
				}
			}
		}
	}

	fmt.Printf("\nNumber of objects: %d\n", len(objects))
	fmt.Printf("\nNumber of definitions: %d\n", numDefs)

	if !*defsOnly {
		fmt.Printf("\nNumber of counters: %d\n", numCounters)
	}
}
