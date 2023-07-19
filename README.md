# perflib_exporter

perflib_exporter and the perflib library are unmaintained
since the author is no longer running any Windows servers.

The perflib library has been
[integrated](https://github.com/prometheus-community/windows_exporter/issues/1240) into
[windows_exporter](https://github.com/prometheus-community/windows_exporter) and development continues there.
Both library users of perflib as well as perflib_exporter
users should migrate to windows_exporter, which is actively
maintained and has well-defined metrics.
