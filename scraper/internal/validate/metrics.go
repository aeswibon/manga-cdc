package validate

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var recordsRejected = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "scraper_records_rejected_total",
	Help: "Total scraped records rejected by validation guardrails",
}, []string{"source", "entity", "rule"})

var recordsAccepted = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "scraper_records_accepted_total",
	Help: "Total scraped records accepted by validation guardrails",
}, []string{"source", "entity"})

func RecordReject(source, entity string, issues []Issue) {
	for _, issue := range issues {
		recordsRejected.WithLabelValues(source, entity, issue.Rule).Inc()
	}
}

func RecordAccept(source, entity string) {
	recordsAccepted.WithLabelValues(source, entity).Inc()
}
