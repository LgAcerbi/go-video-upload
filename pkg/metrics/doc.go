// Package metrics provides a writer that sends tags and fields to InfluxDB without blocking.
//
// How Go does "fire-and-forget" (like not awaiting a Promise in JS):
//
//   - Go has no Promises. To run something without waiting: start a goroutine with "go fn()".
//   - Record() does "go w.writePoint(...)" so the write runs in the background. The HTTP handler or queue consumer returns immediately.
//   - If the write fails inside that goroutine, we log it and drop the point; the caller never sees the error.
//
// Usage:
//
//	w, _ := metrics.NewWriter(metrics.WriterConfig{
//	    URL: "http://localhost:8086", Token: "my-token", Org: "my-org", Bucket: "metrics",
//	})
//	defer w.Close()
//
//	// From an HTTP handler or RabbitMQ consumer:
//	w.Record("http_request", map[string]string{"path": "/videos", "method": "GET"}, map[string]interface{}{"count": 1, "status": 200})
//
// Grafana: add InfluxDB as a datasource and build panels from the same bucket/measurements.
package metrics
