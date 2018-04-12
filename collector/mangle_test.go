package collector

import (
	"fmt"
	"sort"

	"github.com/leoluk/perflib_exporter/perflib"
)

func ExampleMakePrometheusLabel() {
	objects, err := perflib.QueryPerformanceData("Global")

	if err != nil {
		panic(err)
	}

	testIndices := []uint{
		10,   // File Read Operations/sec
		44,   // Processor Queue Length
		1350, // % Registry Quota In Use
		1676, // Free & Zero Page List Bytes
		94,   // Data Map Hits %
		228,  // Avg. Disk Bytes/Write
		388,  // Bytes Total/sec
		1260, // Logon Total
		1262, // Total Durable Handles
		4412, // Total Failed Persistent Handle Reopen Count
		4552, // Response Time -- Minimum
		4622, // # of resumed workflow jobs/sec
		206,  // Avg. Disk sec/Transfer
	}

	sort.Slice(testIndices, func(i, j int) bool {
		return testIndices[i] < testIndices[j]
	})

	names := make(map[uint]string)

	for _, o := range objects {
		for _, d := range o.CounterDefs {
			for _, n := range testIndices {
				if d.NameIndex == n {
					names[n] = MakePrometheusLabel(d)
				}
			}
		}
	}

	for _, n := range testIndices {
		fmt.Printf("%d %s\n", n, names[n])
	}

/* Output:
10 file_read_operations_total
44 processor_queue_length
94 data_map_hits_total
206 avg_disk_sec_per_transfer_total
228 avg_disk_bytes_per_write_total
388 bytes_total
1260 logon
1262 durable_handles
1350 registry_quota_in_use_total
1676 free_and_zero_page_list_bytes
4412 failed_persistent_handle_reopen_count
4552 response_time_minimum
4622 resumed_workflow_jobs_total
*/

}
