package main

import (
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

// ZoneExporter collects metrics for a Cloudflare zone.
type ZoneExporter struct {
	cf    *cloudflare.API
	zone  cloudflare.Zone
	colos map[string]colo

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

// NewZoneExporter returns an initialized ZoneExporter.
func NewZoneExporter(api *cloudflare.API, coloMap map[string]colo, zone cloudflare.Zone) *ZoneExporter {
	defaultLabels := []string{"colo_id", "colo_name", "colo_region"}

	return &ZoneExporter{
		cf:    api,
		zone:  zone,
		colos: coloMap,
		allRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "total"),
			"Total number of requests served",
			defaultLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		cachedRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "cached"),
			"Total number of cached requests served",
			defaultLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		uncachedRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "uncached"),
			"Total number of requests served from the origin",
			defaultLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		encryptedRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "encrypted"),
			"The number of requests served over HTTPS",
			defaultLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		unencryptedRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "unencrypted"),
			"The number of requests served over HTTP",
			defaultLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		byStatusRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "by_status"),
			"The total number of requests broken out by status code",
			append(defaultLabels, "status_code"),
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		byContentTypeRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "by_content_type"),
			"The total number of requests broken out by content type",
			append(defaultLabels, "content_type"),
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		byCountryRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "by_country"),
			"The total number of requests broken out by country",
			append(defaultLabels, "country_code"),
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		byIPClassRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "by_ip_class"),
			"The total number of requests broken out by IP class",
			append(defaultLabels, "ip_class"),
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),

		totalBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bandwidth", "total_bytes"),
			"The total number of bytes served within the time frame",
			defaultLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		cachedBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bandwidth", "cached_bytes"),
			"The total number of bytes that were cached (and served) by Cloudflare",
			defaultLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		uncachedBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bandwidth", "uncached_bytes"),
			"The total number of bytes that were fetched and served from the origin server",
			defaultLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		encryptedBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bandwidth", "encrypted_bytes"),
			"The total number of bytes served over HTTPS",
			defaultLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		unencryptedBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bandwidth", "unencrypted_bytes"),
			"The total number of bytes served over HTTP",
			defaultLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		byContentTypeBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bandwidth", "by_content_type_bytes"),
			"The total number of bytes served broken out by content type",
			append(defaultLabels, "content_type"),
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		byCountryBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bandwidth", "by_country_bytes"),
			"The total number of bytes served broken out by country",
			append(defaultLabels, "country_code"),
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),

		allThreats: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "threats", "total"),
			"The total number of identifiable threats received",
			defaultLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		byTypeThreats: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "threats", "by_type"),
			"The total number of identifiable threats received broken out by type",
			append(defaultLabels, "type"),
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		byCountryThreats: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "threats", "by_country"),
			"The total number of identifiable threats received broken out by country",
			append(defaultLabels, "country_code"),
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),

		allPageviews: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pageviews", "total"),
			"The total number of pageviews served",
			defaultLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		bySearchEnginePageviews: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pageviews", "by_search_engine"),
			"The total number of pageviews served broken out by search engine",
			append(defaultLabels, "search_engine"),
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),

		uniqueIPAddresses: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "unique_ip_addresses", "total"),
			"Total number of unique IP addresses",
			defaultLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),

		dnsQueryTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "dns_record", "queries_total"),
			"Total number of DNS queries",
			[]string{"query_name", "response_code", "origin", "tcp", "ip_version", "colo_id", "colo_name", "colo_region", "query_type"},
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		uncachedDNSQueries: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "dns_record", "uncached_queries_total"),
			"Total number of uncached DNS queries",
			[]string{"query_name", "response_code", "origin", "tcp", "ip_version", "colo_id", "colo_name", "colo_region", "query_type"},
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		staleDNSQueries: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "dns_record", "stale_queries_total"),
			"Total number of DNS queries",
			[]string{"query_name", "response_code", "origin", "tcp", "ip_version", "colo_id", "colo_name", "colo_region", "query_type"},
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
	}
}

// Describe describes all the metrics exported by the cloudflare ZoneExporter. It
// implements prometheus.Collector.
func (e *ZoneExporter) Describe(ch chan<- *prometheus.Desc) {
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

// Collect fetches the statistics for the configured Cloudflare zone, and
// delivers them as Prometheus metrics. It implements prometheus.Collector.
func (e *ZoneExporter) Collect(ch chan<- prometheus.Metric) {
	log.Debugf("Getting data for zone %s (%s)", e.zone.Name, e.zone.ID)
	e.getDashboardAnalytics(ch, e.zone)
	e.getDNSAnalytics(ch, e.zone)
}

func (e *ZoneExporter) getDashboardAnalytics(ch chan<- prometheus.Metric, z cloudflare.Zone) {
	now := time.Now().UTC()
	sinceTime := now.Add(-10080 * time.Minute) // 7 days
	if e.zone.Plan.Price > 200 {
		sinceTime = now.Add(-30 * time.Minute) // Anything higher than business gets 1 minute resolution, minimum -30 minutes
	} else if e.zone.Plan.Price == 200 {
		sinceTime = now.Add(-6 * time.Hour) // Business plans get 15 minute resolution, minimum -6 hours
	} else if e.zone.Plan.Price == 20 {
		sinceTime = now.Add(-24 * time.Hour) // Pro plans get 15 minute resolution, minimum -24 hours
	}
	continuous := false
	opts := cloudflare.ZoneAnalyticsOptions{
		Since:      &sinceTime,
		Until:      &now,
		Continuous: &continuous,
	}
	var data []cloudflare.ZoneAnalyticsData
	var err error
	if e.zone.Plan.Price > 200 {
		data, err = e.cf.ZoneAnalyticsByColocation(e.zone.ID, opts)
	} else {
		singleData, singleDataErr := e.cf.ZoneAnalyticsDashboard(e.zone.ID, opts)
		err = singleDataErr
		data = append(data, singleData)
	}
	if err != nil {
		log.Errorf("failed to get dashboard analytics from cloudflare for zone %s: %s", e.zone.Name, err)
		return
	}

	for _, entry := range data {
		labels := []string{}

		coloID := ""
		coloName := ""
		coloRegion := ""
		if e.zone.Plan.Price > 200 && entry.ColocationID != "" {
			colo := e.getColo(entry.ColocationID)
			coloID = colo.Code
			coloName = colo.Name
			coloRegion = colo.Region
		}

		labels = append(labels, coloID, coloName, coloRegion)

		ch <- prometheus.MustNewConstMetric(e.allRequests, prometheus.CounterValue, float64(entry.Totals.Requests.All), labels...)
		ch <- prometheus.MustNewConstMetric(e.cachedRequests, prometheus.CounterValue, float64(entry.Totals.Requests.Cached), labels...)
		ch <- prometheus.MustNewConstMetric(e.uncachedRequests, prometheus.CounterValue, float64(entry.Totals.Requests.Uncached), labels...)
		ch <- prometheus.MustNewConstMetric(e.encryptedRequests, prometheus.CounterValue, float64(entry.Totals.Requests.SSL.Encrypted), labels...)
		ch <- prometheus.MustNewConstMetric(e.unencryptedRequests, prometheus.CounterValue, float64(entry.Totals.Requests.SSL.Unencrypted), labels...)
		for code, count := range entry.Totals.Requests.HTTPStatus {
			ch <- prometheus.MustNewConstMetric(e.byStatusRequests, prometheus.CounterValue, float64(count), append(labels, code)...)
		}
		for contentType, count := range entry.Totals.Requests.ContentType {
			ch <- prometheus.MustNewConstMetric(e.byContentTypeRequests, prometheus.CounterValue, float64(count), append(labels, contentType)...)
		}
		for country, count := range entry.Totals.Requests.Country {
			ch <- prometheus.MustNewConstMetric(e.byCountryRequests, prometheus.CounterValue, float64(count), append(labels, country)...)
		}
		for class, count := range entry.Totals.Requests.IPClass {
			ch <- prometheus.MustNewConstMetric(e.byIPClassRequests, prometheus.CounterValue, float64(count), append(labels, class)...)
		}

		ch <- prometheus.MustNewConstMetric(e.totalBandwidth, prometheus.GaugeValue, float64(entry.Totals.Bandwidth.All), labels...)
		ch <- prometheus.MustNewConstMetric(e.cachedBandwidth, prometheus.GaugeValue, float64(entry.Totals.Bandwidth.Cached), labels...)
		ch <- prometheus.MustNewConstMetric(e.uncachedBandwidth, prometheus.GaugeValue, float64(entry.Totals.Bandwidth.Uncached), labels...)
		ch <- prometheus.MustNewConstMetric(e.encryptedBandwidth, prometheus.GaugeValue, float64(entry.Totals.Bandwidth.SSL.Encrypted), labels...)
		ch <- prometheus.MustNewConstMetric(e.unencryptedBandwidth, prometheus.GaugeValue, float64(entry.Totals.Bandwidth.SSL.Unencrypted), labels...)
		for contentType, count := range entry.Totals.Bandwidth.ContentType {
			ch <- prometheus.MustNewConstMetric(e.byContentTypeBandwidth, prometheus.GaugeValue, float64(count), append(labels, contentType)...)
		}
		for country, count := range entry.Totals.Bandwidth.Country {
			ch <- prometheus.MustNewConstMetric(e.byCountryBandwidth, prometheus.GaugeValue, float64(count), append(labels, country)...)
		}

		ch <- prometheus.MustNewConstMetric(e.allThreats, prometheus.GaugeValue, float64(entry.Totals.Threats.All), labels...)
		for threatType, count := range entry.Totals.Threats.Type {
			ch <- prometheus.MustNewConstMetric(e.byTypeThreats, prometheus.GaugeValue, float64(count), append(labels, threatType)...)
		}
		for country, count := range entry.Totals.Threats.Country {
			ch <- prometheus.MustNewConstMetric(e.byCountryThreats, prometheus.GaugeValue, float64(count), append(labels, country)...)
		}

		ch <- prometheus.MustNewConstMetric(e.allPageviews, prometheus.GaugeValue, float64(entry.Totals.Pageviews.All), labels...)
		for searchEngine, count := range entry.Totals.Pageviews.SearchEngines {
			ch <- prometheus.MustNewConstMetric(e.bySearchEnginePageviews, prometheus.GaugeValue, float64(count), append(labels, searchEngine)...)
		}

		ch <- prometheus.MustNewConstMetric(e.uniqueIPAddresses, prometheus.GaugeValue, float64(entry.Totals.Uniques.All), labels...)
	}
}

func (e *ZoneExporter) getDNSAnalytics(ch chan<- prometheus.Metric, z cloudflare.Zone) {
	now := time.Now().UTC()
	sinceTime := now.Add(-1 * time.Minute)
	dimensions := []string{"queryName", "responseCode", "origin", "tcp", "ipVersion"}
	if e.zone.Plan.Price >= 200 { // Business plans
		dimensions = []string{"queryName", "responseCode", "origin", "tcp", "ipVersion", "coloName", "queryType"}
	} else if e.zone.Plan.Price == 20 {
		dimensions = []string{"queryName", "responseCode", "origin", "tcp", "ipVersion", "coloName"}
	}
	data, err := e.cf.ZoneDNSAnalytics(e.zone.ID, cloudflare.ZoneDNSAnalyticsOptions{
		Since:      &sinceTime,
		Until:      &now,
		Metrics:    []string{"queryCount", "uncachedCount", "staleCount"},
		Dimensions: dimensions,
	})
	if err != nil {
		log.Errorf("failed to get dns analytics from cloudflare for zone %s: %s", e.zone.Name, err)
		return
	}

	for _, row := range data.Rows {
		queryCount := row.Metrics[0]
		uncachedCount := row.Metrics[1]
		staleCount := row.Metrics[2]

		labels := []string{row.Dimensions[0], row.Dimensions[1], row.Dimensions[2], row.Dimensions[3], row.Dimensions[4]}

		coloID := ""
		coloName := ""
		coloRegion := ""
		if len(row.Dimensions) >= 6 {
			colo := e.getColo(row.Dimensions[5])
			coloID = colo.Code
			coloName = colo.Name
			coloRegion = colo.Region
		}

		labels = append(labels, coloID, coloName, coloRegion)

		queryType := ""
		if len(row.Dimensions) == 7 { // coloName AND queryType
			queryType = row.Dimensions[6]
		}
		labels = append(labels, queryType)

		for idx, dim := range row.Dimensions {
			log.Debugf("%s Dimension %d: %s", e.zone.Name, idx, dim)
		}
		for idx, label := range labels {
			log.Debugf("%s Dimension %d: %s", e.zone.Name, idx, label)
		}

		ch <- prometheus.MustNewConstMetric(e.dnsQueryTotal, prometheus.GaugeValue, queryCount, labels...)
		ch <- prometheus.MustNewConstMetric(e.uncachedDNSQueries, prometheus.GaugeValue, uncachedCount, labels...)
		ch <- prometheus.MustNewConstMetric(e.staleDNSQueries, prometheus.GaugeValue, staleCount, labels...)
	}
}

func (e *ZoneExporter) getColo(coloID string) colo {
	// if coloID == "SJC-PIG" {
	// 	coloID = "SJC"
	// }
	return e.colos[coloID]
}
