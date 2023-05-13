package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/leoluk/perflib_exporter/perflib"
)

type objectTimings struct {
	index uint
	name  string
	value time.Duration
}

func main() {
	args := kingpin.Arg("query", "Perflib query").Required().Strings()
	kingpin.Parse()
	query := strings.Join(*args, " ")

	objects, err := perflib.QueryPerformanceData(query)

	if err != nil {
		panic(err)
	}

	objectIndices := make([]uint, len(objects))
	objectNames := make(map[uint]string)

	for i, o := range objects {
		objectIndices[i] = o.NameIndex
		objectNames[o.NameIndex] = o.Name
	}

	timings := make([]objectTimings, len(objectIndices))
	providerGroups := make([][]uint, len(objectIndices))

	for i, v := range objectIndices {
		tStart := time.Now()
		objects2, err := perflib.QueryPerformanceData(strconv.Itoa(int(v)))
		if err != nil {
			panic(err)
		}
		tEnd := time.Now()
		queryTime := tEnd.Sub(tStart)
		timings[i] = objectTimings{
			index: v,
			name:  objectNames[v],
			value: queryTime,
		}

		for _, o := range objects2 {
			providerGroups[i] = append(providerGroups[i], o.NameIndex)
		}
	}

	sort.Slice(timings, func(i, j int) bool {
		return timings[i].value > timings[j].value
	})

	for _, v := range timings {
		bar := strings.Repeat("█", int(v.value/100000/2))
		fmt.Printf("%s %d %s %s\n", bar, v.index, v.name, v.value)
	}

	fmt.Println("\nProvider groups:")

	for i, v := range providerGroups {
		n := objectIndices[i]
		fmt.Println("-", n, objectNames[n])

		for _, w := range v {
			fmt.Println("   -->", w, objectNames[w])
		}
	}
}
