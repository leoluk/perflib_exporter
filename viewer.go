// +build windows

package main

import (
	"html/template"
	"net/http"
	"time"

	"github.com/leoluk/perflib_exporter/collector"
	"github.com/leoluk/perflib_exporter/perflib"
)

type dumpHandlerTpl struct {
	Objects   *[]*perflib.PerfObject
	Query     string
	QueryTime time.Duration
	Count     int
}

func queryFromRequest(r *http.Request, defaultVal string) string {
	if val, ok := r.URL.Query()["query"]; ok {
		if val[0] == "_default_" {
			defaultVal = defaultQuery
		} else {
			defaultVal = val[0]
		}
	}

	return defaultVal
}

func dumpHandler(w http.ResponseWriter, r *http.Request) {
	query := queryFromRequest(r, "Global")

	// TODO: document params

	tStart := time.Now()
	objects, err := perflib.QueryPerformanceData(query)
	tEnd := time.Now()
	queryTime := tEnd.Sub(tStart)

	count := 0

	for _, o := range objects {
		for _, i := range o.Instances {
			for range i.Counters {
				count += 1
			}
		}
	}

	data := dumpHandlerTpl{
		Query:     query,
		QueryTime: queryTime,
		Count:     count,
	}

	if _, ok := r.URL.Query()["no_sort"]; !ok {
		perflib.SortObjects(objects)
	}

	data.Objects = &objects

	if err != nil {
		panic(err)
	}

	t := template.New("dump").Funcs(template.FuncMap{
		"mangle":     collector.MakePrometheusLabel,
		"has_labels": collector.HasPromotedLabels,
		"labels": func(n uint, instance *perflib.PerfInstance) map[string]string {
			m := make(map[string]string)
			labels := collector.PromotedLabelsForObject(n)
			values := collector.PromotedLabelValuesForInstance(n, instance)

			for i, v := range labels {
				m[v] = values[i]
			}

			return m
		},
	})

	t, err = t.Parse(`<!DOCTYPE html>
	<html lang="en">
	<head>
	    <meta charset="UTF-8">
	    <title>perflib_exporter dump</title>
	</head>
	<body>
	<h1>perflib_exporter dump</h1>
	
	<p>Query: {{ .Query }}</p>
	<p>Object count: {{ .Objects | len }}</p>
	<p>Metric count: {{ .Count }}</p>
	<p>Query duration: {{ .QueryTime }}</p>
	
	<ul>
	{{ range .Objects }}
		<li><a href="#{{ .NameIndex }}">[{{ .NameIndex }}] {{ .Name }}</a></li>
	{{ end }}
	</ul>
	
	{{ range .Objects }}
	<h3 id="{{ .NameIndex }}">[{{ .NameIndex }}] {{ .Name }}</h3>
	
	<table border="1">
	    <tr>
	        <th>Name</th>
	        <th>Mangled name</th>
	        <th>Type</th>
	        <th>IsCounter</th>
	        <th>IsNsCtr</th>
	        <th>Value</th>
	        <th>Help Text</th>
	    </tr>
	    {{ with index .Instances 0 }}
	    {{ range .Counters }}
	    <tr>
	    	{{ with .Def }}
	        <td>[{{ .NameIndex  }}] {{ .Name }}</td>
	        <td>{{ . | mangle }}</td>
	        <td>0x{{ .CounterType | printf "%x" }}</td>
	        <td>{{ .IsCounter }}</td>
	        <td>{{ .IsNanosecondCounter }}</td>
	        {{ end }}
	        <td>{{ .Value }}</td>
	        <td>{{ .Def.HelpText }}</td>
	    </tr>
	    {{ end }}
	    {{ end }}
	</table>
	<p></p>
	
	{{ $num := len .Instances }}
	{{ $objIdx := .NameIndex }}
	{{ $hasLabels := has_labels $objIdx }}
	{{ if gt $num 1 }}
	Instances:
	
	<ul>
	    {{ range .Instances }}
	    {{ if $hasLabels }}
	    <li>name=<b>{{ .Name }}</b>
	    {{ range $k, $v := labels $objIdx . }}
	    {{ $k }}={{ $v }}
	    {{ end }} 
	    </li>
	    {{ else }}
	    <li>{{ .Name }}</li>
	    {{ end }}
	    {{ end }}
	</ul>
	{{ end }}
	{{ end }}
	</body>
	</html>`)

	if err != nil {
		panic(err)
	}

	t.Execute(w, data)
}
