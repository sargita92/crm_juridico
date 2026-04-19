package infrastructure

import "github.com/prometheus/client_golang/prometheus"

// StoredTotal counts persisted files broken down by media type and direction.
//   media_type ∈ {image, document, audio, video, other}
//   direction  ∈ {inbound, outbound}
var StoredTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "crm",
		Subsystem: "files",
		Name:      "stored_total",
		Help:      "Total files captured (persisted) by media type and direction.",
	},
	[]string{"media_type", "direction"},
)

// DownloadsTotal counts successful file downloads, broken down by media type.
var DownloadsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "crm",
		Subsystem: "files",
		Name:      "downloads_total",
		Help:      "Total successful file downloads, by media type.",
	},
	[]string{"media_type"},
)

// StoredBytes tracks the cumulative bytes captured. Cheap monotonic gauge that
// lets operators eyeball storage growth without running du on disk.
var StoredBytes = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: "crm",
		Subsystem: "files",
		Name:      "stored_bytes_total",
		Help:      "Total bytes of media captured since startup.",
	},
)

func init() {
	prometheus.MustRegister(StoredTotal, DownloadsTotal, StoredBytes)
}
