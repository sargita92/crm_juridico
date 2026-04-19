package infrastructure

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestStoredTotal_CountsByTypeAndDirection(t *testing.T) {
	StoredTotal.Reset()
	StoredTotal.WithLabelValues("image", "inbound").Inc()
	StoredTotal.WithLabelValues("image", "inbound").Inc()
	StoredTotal.WithLabelValues("document", "inbound").Inc()
	StoredTotal.WithLabelValues("audio", "outbound").Inc()

	assert.Equal(t, float64(2), testutil.ToFloat64(StoredTotal.WithLabelValues("image", "inbound")))
	assert.Equal(t, float64(1), testutil.ToFloat64(StoredTotal.WithLabelValues("document", "inbound")))
	assert.Equal(t, float64(1), testutil.ToFloat64(StoredTotal.WithLabelValues("audio", "outbound")))
}

func TestDownloadsTotal_CountsByType(t *testing.T) {
	DownloadsTotal.Reset()
	DownloadsTotal.WithLabelValues("image").Inc()
	DownloadsTotal.WithLabelValues("image").Inc()
	DownloadsTotal.WithLabelValues("document").Inc()

	assert.Equal(t, float64(2), testutil.ToFloat64(DownloadsTotal.WithLabelValues("image")))
	assert.Equal(t, float64(1), testutil.ToFloat64(DownloadsTotal.WithLabelValues("document")))
}
