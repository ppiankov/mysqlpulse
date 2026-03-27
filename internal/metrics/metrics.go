package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	MySQLUp = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_up",
		Help: "Whether MySQL is reachable (1=up, 0=down).",
	}, []string{"instance"})

	ScrapeDuration = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_scrape_duration_seconds",
		Help: "Duration of the last scrape in seconds.",
	}, []string{"instance"})

	ScrapeErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "mysql_scrape_errors_total",
		Help: "Total number of scrape errors.",
	}, []string{"instance"})
)

// Register adds all metric descriptors to the given registerer.
func Register(reg prometheus.Registerer) {
	reg.MustRegister(MySQLUp, ScrapeDuration, ScrapeErrors)
}
