package main

import (
	"net/http"
	"os"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

const (
	namespace = "cloudflare"
)

// Exporter collects metrics for a Cloudflare zone.
type Exporter struct {
	cf *cloudflare.API

	allRequests      *prometheus.Desc
	cachedRequests   *prometheus.Desc
	uncachedRequests *prometheus.Desc

	encryptedRequests   *prometheus.Desc
	unencryptedRequests *prometheus.Desc

	byStatusRequests      *prometheus.Desc
	byContentTypeRequests *prometheus.Desc
	byCountryRequests     *prometheus.Desc
	byIPClassRequests     *prometheus.Desc

	totalBandwidth    *prometheus.Desc
	cachedBandwidth   *prometheus.Desc
	uncachedBandwidth *prometheus.Desc

	encryptedBandwidth   *prometheus.Desc
	unencryptedBandwidth *prometheus.Desc

	byContentTypeBandwidth *prometheus.Desc
	byCountryBandwidth     *prometheus.Desc

	allThreats       *prometheus.Desc
	byTypeThreats    *prometheus.Desc
	byCountryThreats *prometheus.Desc

	allPageviews            *prometheus.Desc
	bySearchEnginePageviews *prometheus.Desc

	uniqueIPAddresses *prometheus.Desc

	dnsQueryTotal      *prometheus.Desc
	uncachedDNSQueries *prometheus.Desc
	staleDNSQueries    *prometheus.Desc
}

// NewExporter returns an initialized exporter.
func NewExporter(cfAPI *cloudflare.API) *Exporter {
	return &Exporter{
		cf: cfAPI,
		allRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "total"),
			"Total number of requests served",
			[]string{"zone_id", "zone_name"},
			nil,
		),
		cachedRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "cached"),
			"Total number of cached requests served",
			[]string{"zone_id", "zone_name"},
			nil,
		),
		uncachedRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "uncached"),
			"Total number of requests served from the origin",
			[]string{"zone_id", "zone_name"},
			nil,
		),
		encryptedRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "encrypted"),
			"The number of requests served over HTTPS",
			[]string{"zone_id", "zone_name"},
			nil,
		),
		unencryptedRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "unencrypted"),
			"The number of requests served over HTTP",
			[]string{"zone_id", "zone_name"},
			nil,
		),
		byStatusRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "by_status"),
			"The total number of requests broken out by status code",
			[]string{"zone_id", "zone_name", "status_code"},
			nil,
		),
		byContentTypeRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "by_content_type"),
			"The total number of requests broken out by content type",
			[]string{"zone_id", "zone_name", "content_type"},
			nil,
		),
		byCountryRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "by_country"),
			"The total number of requests broken out by country",
			[]string{"zone_id", "zone_name", "country_code"},
			nil,
		),
		byIPClassRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "by_ip_class"),
			"The total number of requests broken out by IP class",
			[]string{"zone_id", "zone_name", "ip_class"},
			nil,
		),

		totalBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bandwidth", "total"),
			"The total number of bytes served within the time frame",
			[]string{"zone_id", "zone_name"},
			nil,
		),
		cachedBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bandwidth", "cached"),
			"The total number of bytes that were cached (and served) by Cloudflare",
			[]string{"zone_id", "zone_name"},
			nil,
		),
		uncachedBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bandwidth", "uncached"),
			"The total number of bytes that were fetched and served from the origin server",
			[]string{"zone_id", "zone_name"},
			nil,
		),
		encryptedBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bandwidth", "encrypted"),
			"The total number of bytes served over HTTPS",
			[]string{"zone_id", "zone_name"},
			nil,
		),
		unencryptedBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bandwidth", "unencrypted"),
			"The total number of bytes served over HTTP",
			[]string{"zone_id", "zone_name"},
			nil,
		),
		byContentTypeBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bandwidth", "by_content_type"),
			"The total number of bytes served broken out by content type",
			[]string{"zone_id", "zone_name", "content_type"},
			nil,
		),
		byCountryBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bandwidth", "by_country"),
			"The total number of bytes served broken out by country",
			[]string{"zone_id", "zone_name", "country_code"},
			nil,
		),

		allThreats: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "threats", "total"),
			"The total number of identifiable threats received",
			[]string{"zone_id", "zone_name"},
			nil,
		),
		byTypeThreats: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "threats", "by_type"),
			"The total number of identifiable threats received broken out by type",
			[]string{"zone_id", "zone_name", "type"},
			nil,
		),
		byCountryThreats: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "threats", "by_country"),
			"The total number of identifiable threats received broken out by country",
			[]string{"zone_id", "zone_name", "country_code"},
			nil,
		),

		allPageviews: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pageviews", "total"),
			"The total number of pageviews served",
			[]string{"zone_id", "zone_name"},
			nil,
		),
		bySearchEnginePageviews: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pageviews", "by_search_engine"),
			"The total number of pageviews served broken out by search engine",
			[]string{"zone_id", "zone_name", "search_engine"},
			nil,
		),

		uniqueIPAddresses: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "uniques", "total"),
			"Total number of unique IP addresses",
			[]string{"zone_id", "zone_name"},
			nil,
		),

		dnsQueryTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "dns", "queries_total"),
			"Total number of DNS queries",
			[]string{"zone_id", "zone_name", "query_name", "response_code", "origin", "tcp", "ip_version", "colo_name", "query_type"},
			nil,
		),
		uncachedDNSQueries: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "dns", "uncached_queries_total"),
			"Total number of uncached DNS queries",
			[]string{"zone_id", "zone_name", "query_name", "response_code", "origin", "tcp", "ip_version", "colo_name", "query_type"},
			nil,
		),
		staleDNSQueries: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "dns", "stale_queries_total"),
			"Total number of DNS queries",
			[]string{"zone_id", "zone_name", "query_name", "response_code", "origin", "tcp", "ip_version", "colo_name", "query_type"},
			nil,
		),
	}
}

// Describe describes all the metrics exported by the cloudflare exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.allRequests
	ch <- e.cachedRequests
	ch <- e.uncachedRequests
	ch <- e.encryptedRequests
	ch <- e.unencryptedRequests
	ch <- e.byStatusRequests
	ch <- e.byContentTypeRequests
	ch <- e.byCountryRequests
	ch <- e.byIPClassRequests

	ch <- e.totalBandwidth
	ch <- e.cachedBandwidth
	ch <- e.uncachedBandwidth
	ch <- e.encryptedBandwidth
	ch <- e.unencryptedBandwidth
	ch <- e.byContentTypeBandwidth
	ch <- e.byCountryBandwidth

	ch <- e.allThreats
	ch <- e.byTypeThreats
	ch <- e.byCountryThreats

	ch <- e.allPageviews
	ch <- e.bySearchEnginePageviews

	ch <- e.uniqueIPAddresses

	ch <- e.dnsQueryTotal
	ch <- e.uncachedDNSQueries
	ch <- e.staleDNSQueries
}

// Collect fetches the statistics from the configured cloudflare server, and
// delivers them as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	zones, err := e.cf.ListZones()
	if err != nil {
		log.Fatal(err)
	}

	for _, z := range zones {
		e.getDashboardAnalytics(ch, z)
		e.getDNSAnalytics(ch, z)
	}

}

func (e *Exporter) getDashboardAnalytics(ch chan<- prometheus.Metric, z cloudflare.Zone) {
	sinceTime := time.Now().Add(-10080 * time.Minute).UTC() // 7 days
	if z.Plan.Price > 200 {
		sinceTime = time.Now().Add(-30 * time.Minute).UTC() // Anything higher than business gets 1 minute resolution, minimum -30 minutes
	} else if z.Plan.Price == 200 {
		sinceTime = time.Now().Add(-6 * time.Hour).UTC() // Business plans get 15 minute resolution, minimum -6 hours
	} else if z.Plan.Price == 20 {
		sinceTime = time.Now().Add(-24 * time.Hour).UTC() // Pro plans get 15 minute resolution, minimum -24 hours
	}
	data, err := e.cf.ZoneAnalyticsDashboard(z.ID, cloudflare.ZoneAnalyticsOptions{
		Since: &sinceTime,
	})
	if err != nil {
		log.Errorf("Failed to get dashboard analytics from Cloudflare for zone %s: %s", z.Name, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(e.allRequests, prometheus.CounterValue, float64(data.Totals.Requests.All), z.ID, z.Name)
	ch <- prometheus.MustNewConstMetric(e.cachedRequests, prometheus.CounterValue, float64(data.Totals.Requests.Cached), z.ID, z.Name)
	ch <- prometheus.MustNewConstMetric(e.uncachedRequests, prometheus.CounterValue, float64(data.Totals.Requests.Uncached), z.ID, z.Name)
	ch <- prometheus.MustNewConstMetric(e.encryptedRequests, prometheus.CounterValue, float64(data.Totals.Requests.SSL.Encrypted), z.ID, z.Name)
	ch <- prometheus.MustNewConstMetric(e.unencryptedRequests, prometheus.CounterValue, float64(data.Totals.Requests.SSL.Unencrypted), z.ID, z.Name)
	for code, count := range data.Totals.Requests.HTTPStatus {
		ch <- prometheus.MustNewConstMetric(e.byStatusRequests, prometheus.CounterValue, float64(count), z.ID, z.Name, code)
	}
	for contentType, count := range data.Totals.Requests.ContentType {
		ch <- prometheus.MustNewConstMetric(e.byContentTypeRequests, prometheus.CounterValue, float64(count), z.ID, z.Name, contentType)
	}
	for country, count := range data.Totals.Requests.Country {
		ch <- prometheus.MustNewConstMetric(e.byCountryRequests, prometheus.CounterValue, float64(count), z.ID, z.Name, country)
	}
	for class, count := range data.Totals.Requests.IPClass {
		ch <- prometheus.MustNewConstMetric(e.byIPClassRequests, prometheus.CounterValue, float64(count), z.ID, z.Name, class)
	}

	ch <- prometheus.MustNewConstMetric(e.totalBandwidth, prometheus.GaugeValue, float64(data.Totals.Bandwidth.All), z.ID, z.Name)
	ch <- prometheus.MustNewConstMetric(e.cachedBandwidth, prometheus.GaugeValue, float64(data.Totals.Bandwidth.Cached), z.ID, z.Name)
	ch <- prometheus.MustNewConstMetric(e.uncachedBandwidth, prometheus.GaugeValue, float64(data.Totals.Bandwidth.Uncached), z.ID, z.Name)
	ch <- prometheus.MustNewConstMetric(e.encryptedBandwidth, prometheus.GaugeValue, float64(data.Totals.Bandwidth.SSL.Encrypted), z.ID, z.Name)
	ch <- prometheus.MustNewConstMetric(e.unencryptedBandwidth, prometheus.GaugeValue, float64(data.Totals.Bandwidth.SSL.Unencrypted), z.ID, z.Name)
	for contentType, count := range data.Totals.Bandwidth.ContentType {
		ch <- prometheus.MustNewConstMetric(e.byContentTypeBandwidth, prometheus.GaugeValue, float64(count), z.ID, z.Name, contentType)
	}
	for country, count := range data.Totals.Bandwidth.Country {
		ch <- prometheus.MustNewConstMetric(e.byCountryBandwidth, prometheus.GaugeValue, float64(count), z.ID, z.Name, country)
	}

	ch <- prometheus.MustNewConstMetric(e.allThreats, prometheus.GaugeValue, float64(data.Totals.Threats.All), z.ID, z.Name)
	for threatType, count := range data.Totals.Threats.Type {
		ch <- prometheus.MustNewConstMetric(e.byTypeThreats, prometheus.GaugeValue, float64(count), z.ID, z.Name, threatType)
	}
	for country, count := range data.Totals.Threats.Country {
		ch <- prometheus.MustNewConstMetric(e.byCountryThreats, prometheus.GaugeValue, float64(count), z.ID, z.Name, country)
	}

	ch <- prometheus.MustNewConstMetric(e.allPageviews, prometheus.GaugeValue, float64(data.Totals.Pageviews.All), z.ID, z.Name)
	for searchEngine, count := range data.Totals.Pageviews.SearchEngine {
		ch <- prometheus.MustNewConstMetric(e.bySearchEnginePageviews, prometheus.GaugeValue, float64(count), z.ID, z.Name, searchEngine)
	}

	ch <- prometheus.MustNewConstMetric(e.uniqueIPAddresses, prometheus.GaugeValue, float64(data.Totals.Uniques.All), z.ID, z.Name)
}

func (e *Exporter) getDNSAnalytics(ch chan<- prometheus.Metric, z cloudflare.Zone) {
	now := time.Now().UTC()
	sinceTime := now.Add(-1 * time.Minute)
	dimensions := []string{"queryName", "responseCode", "origin", "tcp", "ipVersion"}
	if z.Plan.Price >= 200 { // Business plans
		dimensions = []string{"queryName", "responseCode", "origin", "tcp", "ipVersion", "coloName", "queryType"}
	} else if z.Plan.Price == 20 {
		dimensions = []string{"queryName", "responseCode", "origin", "tcp", "ipVersion", "coloName"}
	}
	data, err := e.cf.ZoneDNSAnalytics(z.ID, cloudflare.ZoneDNSAnalyticsOptions{
		Since:      &sinceTime,
		Until:      &now,
		Metrics:    []string{"queryCount", "uncachedCount", "staleCount"},
		Dimensions: dimensions,
	})
	if err != nil {
		log.Errorf("Failed to get DNS analytics from Cloudflare for zone %s: %s", z.Name, err)
		return
	}

	for _, row := range data.Rows {
		queryCount := row.Metrics[0]
		uncachedCount := row.Metrics[1]
		staleCount := row.Metrics[2]

		queryName := row.Dimensions[0]
		responseCode := row.Dimensions[1]
		origin := row.Dimensions[2]
		tcp := row.Dimensions[3]
		ipVersion := row.Dimensions[4]
		coloName := "N/A"
		queryType := "N/A"
		if len(row.Dimensions) >= 6 {
			coloName = row.Dimensions[5]
		}
		if len(row.Dimensions) == 7 {
			queryType = row.Dimensions[6]
		}

		ch <- prometheus.MustNewConstMetric(e.dnsQueryTotal, prometheus.GaugeValue, queryCount, z.ID, z.Name, queryName, responseCode, origin, tcp, ipVersion, coloName, queryType)
		ch <- prometheus.MustNewConstMetric(e.uncachedDNSQueries, prometheus.GaugeValue, uncachedCount, z.ID, z.Name, queryName, responseCode, origin, tcp, ipVersion, coloName, queryType)
		ch <- prometheus.MustNewConstMetric(e.staleDNSQueries, prometheus.GaugeValue, staleCount, z.ID, z.Name, queryName, responseCode, origin, tcp, ipVersion, coloName, queryType)
	}
}

func init() {
	prometheus.MustRegister(version.NewCollector("cloudflare_exporter"))
}

func main() {
	var (
		listenAddress = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9150").String()
		metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
	)
	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("cloudflare_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infoln("Starting cloudflare_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	api, err := cloudflare.New(os.Getenv("CF_API_KEY"), os.Getenv("CF_API_EMAIL"))
	if err != nil {
		log.Fatal(err)
	}

	prometheus.MustRegister(NewExporter(api))

	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
               <head><title>Cloudflare Exporter</title></head>
               <body>
               <h1>Cloudflare Exporter</h1>
               <p><a href='` + *metricsPath + `'>Metrics</a></p>
               </body>
               </html>`))
	})
	log.Infoln("Starting HTTP server on", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))

}
