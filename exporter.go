package main

import (
	"bytes"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"

	"golang.org/x/sys/windows/svc"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/leoluk/perflib_exporter/collector"
	"github.com/leoluk/perflib_exporter/perflib"
)

// PerflibExporter implements the prometheus.Collector interface.
type PerflibExporter struct {
	collectors map[string]collector.Collector
	logger     log.Logger
}

const (
	serviceName = "perflib_exporter"
)

var (
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(collector.Namespace, "exporter", "collector_duration_seconds"),
		"perflib_exporter: Duration of a collection.",
		[]string{"collector"},
		nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(collector.Namespace, "exporter", "collector_success"),
		"perflib_exporter: Whether the collector was successful.",
		[]string{"collector"},
		nil,
	)
)

// Describe sends all the descriptors of the collectors included to
// the provided channel.
func (coll PerflibExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
}

// Collect sends the collected metrics from each of the collectors to
// prometheus. Collect could be called several times concurrently
// and thus its run is protected by a single mutex.
func (coll PerflibExporter) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	wg.Add(len(coll.collectors))
	for name, c := range coll.collectors {
		go func(name string, c collector.Collector) {
			execute(coll.logger, name, c, ch)
			wg.Done()
		}(name, c)
	}
	wg.Wait()
}

func execute(logger log.Logger, name string, c collector.Collector, ch chan<- prometheus.Metric) {
	begin := time.Now()
	err := c.Collect(ch)
	duration := time.Since(begin)
	var success float64

	if err != nil {
		level.Error(logger).Log("msg", "collector failed", "name", name, "duration", duration, "err", err)
		success = 0
	} else {
		level.Debug(logger).Log("msg", "collector succeed", "name", name, "duration", duration)
		success = 1
	}
	ch <- prometheus.MustNewConstMetric(
		scrapeDurationDesc,
		prometheus.GaugeValue,
		duration.Seconds(),
		name,
	)
	ch <- prometheus.MustNewConstMetric(
		scrapeSuccessDesc,
		prometheus.GaugeValue,
		success,
		name,
	)
}

// Make sure we crash instead of consuming inappropriate amounts of memory
// There's no easy way to set a memory limit on Windows.
func initMemoryGuard(l log.Logger) {
	go func() {
		for {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			if m.Alloc > 250000000 { // 250 MB
				level.Error(l).Log("msg", "CRITICAL: Memory leak detected on heap", "bytes", m.Alloc)
				os.Exit(1)
			}

			time.Sleep(5 * time.Second)
		}
	}()
}

// List of perflib objects enabled by default. Each provider maps
// to one or more objects (see perflib godoc)
var defaultPerflibObjects = []uint32{
	4600, // Network QoS Policy [6 counters, 1 instance(s)]
	4826, // Event Tracing for Windows [6 counters, 1 instance(s)]
	4944, // Thermal Zone Information [3 counters, 1 instance(s)]
	4674, // Processor Information [32 counters, 6 instance(s)]
	4352, // SMB Server Shares [45 counters, 4 instance(s)]
	4952, // HTTP Service [6 counters, 1 instance(s)]
	6674, // Power Meter [2 counters, 1 instance(s)]
	6084, // Microsoft Winsock BSP [4 counters, 1 instance(s)]
	1920, // Terminal Services [3 counters, 1 instance(s)]
	1570, // Security System-Wide Statistics [19 counters, 1 instance(s)]
	4536, // Distributed Transaction Coordinator [13 counters, 1 instance(s)]
	236,  // LogicalDisk [34 counters, 3 instance(s)]
	330,  // Server [41 counters, 1 instance(s)]
	86,   // Cache [34 counters, 1 instance(s)]
	238,  // Processor [15 counters, 5 instance(s)]
	4,    // Memory [37 counters, 1 instance(s)]
	260,  // Objects [6 counters, 1 instance(s)]
	700,  // Paging File [4 counters, 1 instance(s)]
	2,    // System [18 counters, 1 instance(s)]
	1814, // NUMA Node Memory [4 counters, 2 instance(s)]
	230,  // Process [28 counters, 83 instance(s)]
	510,  // Network Interface [22 counters, 2 instance(s)]
	546,  // IPv4 [17 counters, 1 instance(s)]
	582,  // ICMP [27 counters, 1 instance(s)]
	638,  // TCPv4 [9 counters, 1 instance(s)]
	658,  // UDPv4 [5 counters, 1 instance(s)]
	548,  // IPv6 [17 counters, 1 instance(s)]
	1534, // ICMPv6 [33 counters, 1 instance(s)]
	1530, // TCPv6 [9 counters, 1 instance(s)]
	1532, // UDPv6 [5 counters, 1 instance(s)]
}

var (
	defaultQuery string
	authTokens   *[]string
)

func main() {
	var (
		listenAddress = kingpin.Flag(
			"telemetry.addr", "host:port for perflib exporter.").Default(":9432").String()
		metricsPath = kingpin.Flag(
			"telemetry.path", "URL path for surfacing collected metrics.").Default("/metrics").String()

		perfObjects = kingpin.Flag(
			"perflib.objects", "List of perflib object indices to queryBuf (defaults to a built-in list)").Uint32List()
		perfObjectsAdd = kingpin.Flag(
			"perflib.objects.add", "List of perflib object indices to add to list").Uint32List()
		perfObjectsRemove = kingpin.Flag(
			"perflib.objects.remove", "List of perflib object indices to remove from list").Uint32List()
		perfObjectsNames = kingpin.Flag(
			"perflib.objects.names", "List of perflib object names to queryBuf").Strings()
		perfObjectsNamesAdd = kingpin.Flag(
			"perflib.objects.names.add", "List of perflib object names to add to list").Strings()
		perfObjectsNamesRemove = kingpin.Flag(
			"perflib.objects.names.remove", "List of perflib object names to remove from list").Strings()
	)

	authTokens = kingpin.Flag(
		"telemetry.auth", "List of valid bearer tokens. Defaults to none (no auth)").Strings()

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)

	//log.AddFlags(kingpin.CommandLine)
	prometheus.MustRegister(version.NewCollector("perflib_exporter"))
	initMemoryGuard(logger)

	//kingpin.Version(version.Print("perflib_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	// Prepare perflib queryBuf
	var queryBuf bytes.Buffer

	// Get all existing objects if one of the perflib.objects.names flags was used
	var objects []*perflib.PerfObject
	if len(*perfObjectsNames) > 0 || len(*perfObjectsNamesAdd) > 0 || len(*perfObjectsNamesRemove) > 0 {
		o, err := perflib.QueryPerformanceData("Global")
		if err != nil {
			panic(err)
		}
		objects = o
	}

	*perfObjects = append(*perfObjects, objectNamesToIndices(perfObjectsNames, objects)...)

	if len(*perfObjects) == 0 {
		perfObjects = &defaultPerflibObjects
	}

	for _, n := range *perfObjectsAdd {
		*perfObjects = append(*perfObjects, n)
	}

	*perfObjects = append(*perfObjects, objectNamesToIndices(perfObjectsNamesAdd, objects)...)
	*perfObjectsRemove = append(*perfObjectsRemove, objectNamesToIndices(perfObjectsNamesRemove, objects)...)

loopPerfObjects:
	for _, n := range *perfObjects {
		for _, r := range *perfObjectsRemove {
			if n == r {
				continue loopPerfObjects
			}
		}

		queryBuf.WriteString(strconv.Itoa(int(n)) + " ")
	}

	defaultQuery = strings.Trim(queryBuf.String(), " ")
	level.Info(logger).Log("perflib_query", defaultQuery)

	// Initialize Windows service, if necessary
	isInteractive, err := svc.IsAnInteractiveSession()
	if err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}

	stopCh := make(chan bool)
	if !isInteractive {
		go svc.Run(serviceName, &perflibExporterService{stopCh: stopCh, logger: logger})
	}

	// Initialize the exporter
	nodeCollector := PerflibExporter{collectors: map[string]collector.Collector{
		"perflib": collector.NewPerflibCollector(logger, defaultQuery),
	}, logger: logger}

	prometheus.MustRegister(nodeCollector)

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/health", healthCheck)
	http.HandleFunc("/dump", dumpHandler)

	level.Info(logger).Log("msg", "Starting perflib exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "context", version.Info())

	go func() {
		level.Info(logger).Log("msg", "starting server", "listenAddress", listenAddress)
		level.Info(logger).Log("msg", "starting server", "listenAddress", listenAddress)
		err := http.ListenAndServe(*listenAddress, nil)
		level.Error(logger).Log("msg", "cannot start perflib exporter", "err", err)
	}()

	for {
		if <-stopCh {
			level.Info(logger).Log("msg", "shutting down perflib exporter")
			break
		}
	}
}

// objectNamesToIndices converts a slice of perflib object Name values to a slice of perflib NameIndex values
func objectNamesToIndices(names *[]string, objectDefinitions []*perflib.PerfObject) (indices []uint32) {
outerloop:
	for _, p := range *names {
		for _, o := range objectDefinitions {
			if p == o.Name {
				indices = append(indices, uint32(o.NameIndex))
				continue outerloop
			}
		}
	}

	return indices
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"status":"ok"}`)
}

func keys(m map[string]collector.Collector) []string {
	ret := make([]string, 0, len(m))
	for key := range m {
		ret = append(ret, key)
	}
	return ret
}

type perflibExporterService struct {
	stopCh chan<- bool
	logger log.Logger
}

func (s *perflibExporterService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				s.stopCh <- true
				break loop
			default:
				level.Error(s.logger).Log("msg", "unexpected control request", "request", c)
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}
