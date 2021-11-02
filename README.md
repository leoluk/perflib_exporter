# perflib_exporter

The **exporter** is beta-quality software. Please test it, 
but only use it production if you're ready to read the code.
The auto-generated metric names are not guaranteed to be stable
and may change in future releases which improve the mangling logic -
make sure to read the release notes before you upgrade.

The **perflib library** has an API stability guarantee and is ready for production.

---

perflib_exporter is a Prometheus exporter for Windows system performance. 
It queries performance data using the low-level HKEY_PERFORMANCE_DATA 
registry API instead of the high-level WMI or PDH interfaces.

The registry API will return metrics for all perflib providers in a single
binary blob that we have to parse ourselves. This makes it very efficient -
querying all metrics (~20-30k) takes about ~40ms or ~300ms with cold cache.
The providers enabled by default take ~10ms to query.

Its raison d'être is a bug in WMI which causes collection times
to spike every ~16 minutes - see https://github.com/martinlindhe/wmi_exporter/issues/89#issuecomment-359211731 for details. 

This repository contains the "perflib" package, which exposes an API for
querying the HKEY_PERFORMANCE_DATA API. See the godoc for more details:

https://godoc.org/github.com/leoluk/perflib_exporter/perflib

## Usage

Install using "go get":

    go get github.com/leoluk/perflib_exporter
    
perflib groups metrics into different groups/providers. Each provider has a
numerical ID. Query performance depends on the number of providers enabled,
with some being more expensive than others.
    
If you run the binary without arguments, it uses a default list of perflib
exporters. You can manually specify a list of providers to enable. This only
enabled the "System" provider:

    perflib_exporter.exe --perflib.objects=2
    
You can view a HTML dump of all available metrics on `/dump`: 

http://127.0.0.1:9432/dump

Filter by provider:

http://127.0.0.1:9432/dump?query=2

## Command line parameters

    usage: perflib_exporter.exe [<flags>]
    
    Flags:
      -h, --help                    Show context-sensitive help (also try
                                    --help-long and --help-man).
          --telemetry.addr=":9432"  host:port for perflib exporter.
          --telemetry.path="/metrics"
                                    URL path for surfacing collected metrics.
          --perflib.objects=PERFLIB.OBJECTS ...
                                    List of perflib object indices to queryBuf
                                    (defaults to a built-in list)
          --perflib.objects.add=PERFLIB.OBJECTS.ADD ...
                                    List of perflib object indices to add to list
          --perflib.objects.remove=PERFLIB.OBJECTS.REMOVE ...
                                    List of perflib object indices to remove from
                                    list
          --perflib.objects.names=PERFLIB.OBJECTS.NAMES ...
                                    List of perflib object names to queryBuf
          --perflib.objects.names.add=PERFLIB.OBJECTS.NAMES.ADD ...
                                    List of perflib object names to add to list
          --perflib.objects.names.remove=PERFLIB.OBJECTS.NAMES.REMOVE ...
                                    List of perflib object names to remove from list
          --telemetry.auth=TELEMETRY.AUTH ...
                                    List of valid bearer tokens. Defaults to none
                                    (no auth)
          --log.level="info"        Only log messages with the given severity or
                                    above. Valid levels: [debug, info, warn, error,
                                    fatal]
          --log.format="logfmt"
                                    Only supports logfmt (default) or json
          --version                 Show application version.

## TBD

- Document which providers have been tested and are known to work well.

- Metric names and labels are auto-generated. We might want to statically
  generate them for better performance and reliability (discussion needed -
  the name mangling could be done statically, with no name tables needed at runtime)
  
  - Object names are very verbose, shorten them?
    (Distributed Transaction Coordinator -> dtc, Security System-Wide Statistics -> security, ...)
  
  - perflib flags are sometimes wrong, incorrectly identifying a counter as a gauge.
    Do we want to leave them like that for consistency, or make a list of exceptions?
    
    - [350] Errors Access Permissions
    - [1828] SMB BranchCache Hash V2 Generation Requests 
    - [1262] Total Durable Handles
    - [1260] Logon Total
    - [4412] Total Failed Persistent Handle Reopen Count
    - ...
    
  - Group multiple related metrics into a single one with labels ("promotion")
    
  - Byte conversion
    - PercentFreeSpace_Base
    
- Deal with metrics like these (both end up as "logon_total"):
    - [692] Logon/sec (counter)
    - [1260] Logon Total (gauge)
    
- Deal with multi-value counters (base values)
  - Include 100ns base timer (and document how to calculate percentages)
  
- Complete the godoc

- Remove _Total instances
  
- List of objects that are officially supported and auto-generated docs

- Document lack of filtering (we retrieve all instances anyway - use
  relabelling if cardinality is a concern)
    
- No tests done on Windows Server 2008 and 2016 beyond basic functionality
  (they have a slightly different set of perflib providers)
 
- The "Global" query is non-deterministic. This is either a bug in our parsing
  code, or in perflib (...)
  
- Bearer token auth (it's hard to run an extra reverse proxy on Windows)

- MSI installer and sensible firewall config (PowerShell examples)

- perflib: has_instances flag

- Comprehensive tests

- CI/automated builds

