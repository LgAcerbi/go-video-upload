package metrics

import (
	"log/slog"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

type Writer struct {
	writeAPI api.WriteAPI
	org      string
	bucket   string
	log      *slog.Logger
	once     sync.Once
}

type WriterConfig struct {
	URL    string
	Token  string
	Org    string
	Bucket string
	Logger *slog.Logger
}

func NewWriter(cfg WriterConfig) (*Writer, error) {
	if cfg.URL == "" || cfg.Token == "" {
		return nil, nil
	}
	client := influxdb2.NewClient(cfg.URL, cfg.Token)
	writeAPI := client.WriteAPI(cfg.Org, cfg.Bucket)
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}
	w := &Writer{writeAPI: writeAPI, org: cfg.Org, bucket: cfg.Bucket, log: log}
	go w.consumeErrors()
	return w, nil
}

func (w *Writer) consumeErrors() {
	for err := range w.writeAPI.Errors() {
		w.log.Error("influxdb write failed", "error", err)
	}
}

func (w *Writer) Record(measurement string, tags map[string]string, fields map[string]interface{}) {
	if w == nil || w.writeAPI == nil {
		return
	}
	go w.writePoint(measurement, tags, fields)
}

func (w *Writer) writePoint(measurement string, tags map[string]string, fields map[string]interface{}) {
	p := write.NewPoint(measurement, tags, fields, time.Now())
	w.writeAPI.WritePoint(p)
}

func (w *Writer) Close() {
	if w == nil || w.writeAPI == nil {
		return
	}
	w.once.Do(func() {
		w.writeAPI.Flush()
	})
}
