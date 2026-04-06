package application

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	messagesReceivedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "whatsapp_messages_received_total",
			Help: "Total number of WhatsApp messages received",
		},
		[]string{"tenant_id"},
	)

	messagesSentTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "whatsapp_messages_sent_total",
			Help: "Total number of WhatsApp messages sent",
		},
		[]string{"tenant_id", "status"},
	)

	messageProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "whatsapp_message_processing_duration_seconds",
			Help:    "Duration of message processing in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"direction"},
	)
)
