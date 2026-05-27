package infrastructure

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// connectionUp reflects the live WhatsApp connection state per tenant
	// (1 = connected, 0 = disconnected/logged out). Useful for alerting on
	// tenants that drop and never recover.
	connectionUp = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "whatsapp_connection_up",
			Help: "WhatsApp connection state per tenant (1 = connected, 0 = disconnected)",
		},
		[]string{"tenant_id"},
	)

	// disconnectTotal counts disconnect/stream events per tenant by reason, so
	// we can tell a benign auto-reconnect blip from a terminal logout/ban.
	disconnectTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "whatsapp_disconnect_total",
			Help: "Total WhatsApp disconnect/stream events per tenant by reason",
		},
		[]string{"tenant_id", "reason"},
	)
)
