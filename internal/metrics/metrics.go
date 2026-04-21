package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	MessagesSent = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "notifier_messages_sent_total",
		Help: "Total number of successfully sent Telegram messages.",
	})

	MessagesFailed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "notifier_messages_failed_total",
		Help: "Total number of failed Telegram message sends.",
	})

	SendDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "notifier_send_duration_seconds",
		Help:    "Duration of Telegram sendMessage API calls.",
		Buckets: prometheus.DefBuckets,
	})
)

func init() {
	prometheus.MustRegister(MessagesSent, MessagesFailed, SendDuration)
}