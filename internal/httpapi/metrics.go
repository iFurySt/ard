package httpapi

import (
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type metricsCollector struct {
	startedAt time.Time
	inFlight  atomic.Int64
	mu        sync.Mutex
	requests  map[metricsKey]uint64
	latency   map[metricsKey]time.Duration
	buckets   map[metricsKey][]uint64
}

type metricsKey struct {
	method string
	route  string
	status int
}

func newMetricsCollector() *metricsCollector {
	return &metricsCollector{
		startedAt: time.Now().UTC(),
		requests:  map[metricsKey]uint64{},
		latency:   map[metricsKey]time.Duration{},
		buckets:   map[metricsKey][]uint64{},
	}
}

func metricsMiddleware(metrics *metricsCollector) gin.HandlerFunc {
	return func(context *gin.Context) {
		startedAt := time.Now()
		metrics.inFlight.Add(1)
		defer metrics.inFlight.Add(-1)

		context.Next()

		metrics.record(
			context.Request.Method,
			routeLabel(context),
			context.Writer.Status(),
			time.Since(startedAt),
		)
	}
}

func (metrics *metricsCollector) record(method string, route string, status int, latency time.Duration) {
	key := metricsKey{method: method, route: route, status: status}
	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	metrics.requests[key]++
	metrics.latency[key] += latency
	if _, ok := metrics.buckets[key]; !ok {
		metrics.buckets[key] = make([]uint64, len(httpRequestDurationBuckets))
	}
	seconds := latency.Seconds()
	for index, bucket := range httpRequestDurationBuckets {
		if seconds <= bucket {
			metrics.buckets[key][index]++
		}
	}
}

func (metrics *metricsCollector) render() string {
	metrics.mu.Lock()
	keys := make([]metricsKey, 0, len(metrics.requests))
	requests := make(map[metricsKey]uint64, len(metrics.requests))
	latency := make(map[metricsKey]time.Duration, len(metrics.latency))
	buckets := make(map[metricsKey][]uint64, len(metrics.buckets))
	for key, value := range metrics.requests {
		keys = append(keys, key)
		requests[key] = value
	}
	for key, value := range metrics.latency {
		latency[key] = value
	}
	for key, value := range metrics.buckets {
		buckets[key] = append([]uint64(nil), value...)
	}
	metrics.mu.Unlock()

	sort.Slice(keys, func(i int, j int) bool {
		if keys[i].method != keys[j].method {
			return keys[i].method < keys[j].method
		}
		if keys[i].route != keys[j].route {
			return keys[i].route < keys[j].route
		}
		return keys[i].status < keys[j].status
	})

	var builder strings.Builder
	builder.WriteString("# HELP ard_registry_uptime_seconds Seconds since the registry process started.\n")
	builder.WriteString("# TYPE ard_registry_uptime_seconds gauge\n")
	fmt.Fprintf(&builder, "ard_registry_uptime_seconds %.3f\n", time.Since(metrics.startedAt).Seconds())
	builder.WriteString("# HELP ard_http_requests_in_flight Current in-flight HTTP requests.\n")
	builder.WriteString("# TYPE ard_http_requests_in_flight gauge\n")
	fmt.Fprintf(&builder, "ard_http_requests_in_flight %d\n", metrics.inFlight.Load())
	writeRuntimeMetrics(&builder)
	builder.WriteString("# HELP ard_http_requests_total Total HTTP requests by method, route, and status.\n")
	builder.WriteString("# TYPE ard_http_requests_total counter\n")
	for _, key := range keys {
		fmt.Fprintf(&builder, "ard_http_requests_total{%s} %d\n", metricLabels(key), requests[key])
	}
	builder.WriteString("# HELP ard_http_request_duration_seconds HTTP request duration by method, route, and status.\n")
	builder.WriteString("# TYPE ard_http_request_duration_seconds histogram\n")
	for _, key := range keys {
		labels := metricLabels(key)
		for index, bucket := range httpRequestDurationBuckets {
			fmt.Fprintf(
				&builder,
				"ard_http_request_duration_seconds_bucket{%s,le=%s} %d\n",
				labels,
				strconv.Quote(formatBucket(bucket)),
				buckets[key][index],
			)
		}
		fmt.Fprintf(&builder, "ard_http_request_duration_seconds_bucket{%s,le=\"+Inf\"} %d\n", labels, requests[key])
		fmt.Fprintf(&builder, "ard_http_request_duration_seconds_sum{%s} %.9f\n", labels, latency[key].Seconds())
		fmt.Fprintf(&builder, "ard_http_request_duration_seconds_count{%s} %d\n", labels, requests[key])
	}
	return builder.String()
}

func writeRuntimeMetrics(builder *strings.Builder) {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	builder.WriteString("# HELP ard_runtime_goroutines Current number of goroutines.\n")
	builder.WriteString("# TYPE ard_runtime_goroutines gauge\n")
	fmt.Fprintf(builder, "ard_runtime_goroutines %d\n", runtime.NumGoroutine())
	builder.WriteString("# HELP ard_runtime_heap_alloc_bytes Bytes allocated and still in use on the heap.\n")
	builder.WriteString("# TYPE ard_runtime_heap_alloc_bytes gauge\n")
	fmt.Fprintf(builder, "ard_runtime_heap_alloc_bytes %d\n", stats.HeapAlloc)
	builder.WriteString("# HELP ard_runtime_heap_sys_bytes Bytes of heap memory obtained from the OS.\n")
	builder.WriteString("# TYPE ard_runtime_heap_sys_bytes gauge\n")
	fmt.Fprintf(builder, "ard_runtime_heap_sys_bytes %d\n", stats.HeapSys)
	builder.WriteString("# HELP ard_runtime_next_gc_bytes Target heap size for the next GC cycle.\n")
	builder.WriteString("# TYPE ard_runtime_next_gc_bytes gauge\n")
	fmt.Fprintf(builder, "ard_runtime_next_gc_bytes %d\n", stats.NextGC)
	builder.WriteString("# HELP ard_runtime_gc_cycles_total Completed GC cycles.\n")
	builder.WriteString("# TYPE ard_runtime_gc_cycles_total counter\n")
	fmt.Fprintf(builder, "ard_runtime_gc_cycles_total %d\n", stats.NumGC)
	builder.WriteString("# HELP ard_runtime_last_gc_unix_seconds Unix timestamp of the last completed GC cycle, or 0 when no GC has completed.\n")
	builder.WriteString("# TYPE ard_runtime_last_gc_unix_seconds gauge\n")
	fmt.Fprintf(builder, "ard_runtime_last_gc_unix_seconds %.9f\n", float64(stats.LastGC)/float64(time.Second))
}

func (server Server) metrics(context *gin.Context) {
	context.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(server.metricsCollector.render()))
}

func routeLabel(context *gin.Context) string {
	if route := context.FullPath(); route != "" {
		return route
	}
	return "unmatched"
}

func metricLabels(key metricsKey) string {
	return fmt.Sprintf(
		`method=%s,route=%s,status=%s`,
		strconv.Quote(key.method),
		strconv.Quote(key.route),
		strconv.Quote(strconv.Itoa(key.status)),
	)
}

var httpRequestDurationBuckets = []float64{
	0.005,
	0.01,
	0.025,
	0.05,
	0.1,
	0.25,
	0.5,
	1,
	2.5,
	5,
	10,
}

func formatBucket(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
