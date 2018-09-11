package main

import (
	"fmt"
	"strings"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

// ZoneExporter collects metrics for a Cloudflare zone.
type ZoneExporter struct {
	cf            *cloudflare.API
	zone          cloudflare.Zone
	dnsDimensions []string

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

	componentProcessingTime *prometheus.Desc
	overallProcessingTime   *prometheus.Desc
}

// NewZoneExporter returns an initialized ZoneExporter.
func NewZoneExporter(api *cloudflare.API, zone cloudflare.Zone) *ZoneExporter {
	dashboardMetricsLabels := []string{}
	dashboardMetricsNamespace := namespace
	dashboardMetricsHelpSuffix := ""

	dnsDimensions := []string{"queryName", "responseCode", "origin", "tcp", "ipVersion"}
	dnsMetricsLabels := []string{"query_name", "response_code", "origin", "tcp", "ip_version"}
	dnsMetricsNamespace := namespace
	dnsMetricsHelpSuffix := ""

	// Free plans:
	// Dashboard Analytics is for Global Cloudflare network
	// Dashboard Analytics Labels are empty
	// Dashboard Analytics Namespace is "cloudflare"
	// DNS Analytics is for Global Cloudflare network
	// DNS Analytics Labels contain query_name, response_code, origin, tcp, ip_version
	// DNS Analytics Dimensions contain queryName, responseCode, origin, tcp, ipVersion
	// DNS Analytics Namespace is "cloudflare"

	// Pro plans:
	// Dashboard Analytics is for Global Cloudflare network
	// Dashboard Analytics Labels are empty
	// Dashboard Analytics Namespace is "cloudflare"
	// DNS Analytics broken out by point of presence (PoP, sometimes also called "colo")
	// DNS Analytics Labels contain query_name, response_code, response_cached, origin, tcp, ip_version, pop_id, pop_name, pop_region
	// DNS Analytics Dimensions contain queryName, responseCode, origin, tcp, ipVersion, responseCached, coloName (really ID, name/region provided by statuspage)
	// DNS Analytics Namespace is "cloudflare_pop"

	// Business plans:
	// Dashboard Analytics is for Global Cloudflare network
	// Dashboard Analytics Labels are empty
	// Dashboard Analytics Namespace is "cloudflare"
	// DNS Analytics broken out by point of presence (PoP, sometimes also called "colo")
	// DNS Analytics Labels contain query_name, response_code, response_cached, origin, tcp, ip_version, query_type, pop_id, pop_name, pop_region
	// DNS Analytics Dimensions contain queryName, responseCode, origin, tcp, ipVersion, responseCached, queryType, coloName (really ID, name/region provided by statuspage)
	// DNS Analytics Namespace is "cloudflare_pop"

	// Enterprise plans:
	// Dashboard Analytics broken out by point of presence (PoP, sometimes also called "colo")
	// Dashboard Analytics Labels are pop_id, pop_name, pop_region
	// Dashboard Analytics Namespace is "cloudflare_pop"
	// DNS Analytics broken out by point of presence (PoP, sometimes also called "colo")
	// DNS Analytics Labels contain query_name, response_code, response_cached, origin, tcp, ip_version, query_type, pop_id, pop_name, pop_region
	// DNS Analytics Dimensions contain queryName, responseCode, origin, tcp, ipVersion, responseCached, queryType, coloName (really ID, name/region provided by statuspage)
	// DNS Analytics Namespace is "cloudflare_pop"

	if zone.Plan.LegacyID == "enterprise" {
		dashboardMetricsHelpSuffix = "(broken out by point of presence (PoP))"
		dashboardMetricsLabels = []string{"pop_id", "pop_name", "pop_region"}
		dashboardMetricsNamespace = fmt.Sprintf("%s_pop", namespace)

		dnsDimensions = []string{"queryName", "responseCode", "origin", "tcp", "ipVersion", "responseCached", "queryType", "coloName"}
		dnsMetricsHelpSuffix = "(broken out by point of presence (PoP))"
		dnsMetricsLabels = []string{"query_name", "response_code", "origin", "tcp", "ip_version", "response_cached", "query_type", "pop_id", "pop_name", "pop_region"}
		dnsMetricsNamespace = fmt.Sprintf("%s_pop", namespace)
	} else if zone.Plan.LegacyID == "business" {
		dnsMetricsNamespace = fmt.Sprintf("%s_pop", namespace)
		dnsDimensions = []string{"queryName", "responseCode", "origin", "tcp", "ipVersion", "responseCached", "queryType", "coloName"}
		dnsMetricsHelpSuffix = "(broken out by point of presence (PoP))"
		dnsMetricsLabels = []string{"query_name", "response_code", "origin", "tcp", "ip_version", "response_cached", "query_type", "pop_id", "pop_name", "pop_region"}
	} else if zone.Plan.LegacyID == "pro" {
		dnsMetricsNamespace = fmt.Sprintf("%s_pop", namespace)
		dnsDimensions = []string{"queryName", "responseCode", "origin", "tcp", "ipVersion", "coloName"}
		dnsMetricsHelpSuffix = "(broken out by point of presence (PoP))"
		dnsMetricsLabels = []string{"query_name", "response_code", "origin", "tcp", "ip_version", "pop_id", "pop_name", "pop_region"}
	}

	log.Debugf("Zone %s (%s) configured with plan %s", zone.Name, zone.ID, zone.Plan.LegacyID)
	log.Debugf("Dashboard metrics namespace: '%s'", dashboardMetricsNamespace)
	log.Debugf("Dashboard metrics labels: '%s'", strings.Join(dashboardMetricsLabels, ", "))

	log.Debugf("DNS metrics namespace: '%s'", dnsMetricsNamespace)
	log.Debugf("DNS metrics labels: '%s'", strings.Join(dnsMetricsLabels, ", "))
	log.Debugf("DNS dimensions: '%s'", strings.Join(dnsDimensions, ", "))

	return &ZoneExporter{
		cf:            api,
		zone:          zone,
		dnsDimensions: dnsDimensions,
		allRequests: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "requests", "total"),
			fmt.Sprintf("Total number of requests served %s", dashboardMetricsHelpSuffix),
			dashboardMetricsLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		cachedRequests: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "requests", "cached"),
			fmt.Sprintf("Total number of cached requests served %s", dashboardMetricsHelpSuffix),
			dashboardMetricsLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		uncachedRequests: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "requests", "uncached"),
			fmt.Sprintf("Total number of requests served from the origin %s", dashboardMetricsHelpSuffix),
			dashboardMetricsLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		encryptedRequests: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "requests", "encrypted"),
			fmt.Sprintf("The number of requests served over HTTPS %s", dashboardMetricsHelpSuffix),
			dashboardMetricsLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		unencryptedRequests: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "requests", "unencrypted"),
			fmt.Sprintf("The number of requests served over HTTP %s", dashboardMetricsHelpSuffix),
			dashboardMetricsLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		byStatusRequests: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "requests", "by_status"),
			fmt.Sprintf("The total number of requests broken out by status code %s", dashboardMetricsHelpSuffix),
			append(dashboardMetricsLabels, "status_code"),
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		byContentTypeRequests: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "requests", "by_content_type"),
			fmt.Sprintf("The total number of requests broken out by content type %s", dashboardMetricsHelpSuffix),
			append(dashboardMetricsLabels, "content_type"),
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		byCountryRequests: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "requests", "by_country"),
			fmt.Sprintf("The total number of requests broken out by country %s", dashboardMetricsHelpSuffix),
			append(dashboardMetricsLabels, "country_code"),
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		byIPClassRequests: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "requests", "by_ip_class"),
			fmt.Sprintf("The total number of requests broken out by IP class %s", dashboardMetricsHelpSuffix),
			append(dashboardMetricsLabels, "ip_class"),
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),

		totalBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "bandwidth", "total_bytes"),
			fmt.Sprintf("The total number of bytes served within the time frame %s", dashboardMetricsHelpSuffix),
			dashboardMetricsLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		cachedBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "bandwidth", "cached_bytes"),
			fmt.Sprintf("The total number of bytes that were cached (and served) by Cloudflare %s", dashboardMetricsHelpSuffix),
			dashboardMetricsLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		uncachedBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "bandwidth", "uncached_bytes"),
			fmt.Sprintf("The total number of bytes that were fetched and served from the origin server %s", dashboardMetricsHelpSuffix),
			dashboardMetricsLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		encryptedBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "bandwidth", "encrypted_bytes"),
			fmt.Sprintf("The total number of bytes served over HTTPS %s", dashboardMetricsHelpSuffix),
			dashboardMetricsLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		unencryptedBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "bandwidth", "unencrypted_bytes"),
			fmt.Sprintf("The total number of bytes served over HTTP %s", dashboardMetricsHelpSuffix),
			dashboardMetricsLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		byContentTypeBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "bandwidth", "by_content_type_bytes"),
			fmt.Sprintf("The total number of bytes served broken out by content type %s", dashboardMetricsHelpSuffix),
			append(dashboardMetricsLabels, "content_type"),
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		byCountryBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "bandwidth", "by_country_bytes"),
			fmt.Sprintf("The total number of bytes served broken out by country %s", dashboardMetricsHelpSuffix),
			append(dashboardMetricsLabels, "country_code"),
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),

		allThreats: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "threats", "total"),
			fmt.Sprintf("The total number of identifiable threats received %s", dashboardMetricsHelpSuffix),
			dashboardMetricsLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		byTypeThreats: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "threats", "by_type"),
			fmt.Sprintf("The total number of identifiable threats received broken out by type %s", dashboardMetricsHelpSuffix),
			append(dashboardMetricsLabels, "type"),
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		byCountryThreats: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "threats", "by_country"),
			fmt.Sprintf("The total number of identifiable threats received broken out by country %s", dashboardMetricsHelpSuffix),
			append(dashboardMetricsLabels, "country_code"),
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),

		allPageviews: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "pageviews", "total"),
			fmt.Sprintf("The total number of pageviews served %s", dashboardMetricsHelpSuffix),
			dashboardMetricsLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		bySearchEnginePageviews: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "pageviews", "by_search_engine"),
			fmt.Sprintf("The total number of pageviews served broken out by search engine %s", dashboardMetricsHelpSuffix),
			append(dashboardMetricsLabels, "search_engine"),
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),

		uniqueIPAddresses: prometheus.NewDesc(
			prometheus.BuildFQName(dashboardMetricsNamespace, "unique_ip_addresses", "total"),
			fmt.Sprintf("Total number of unique IP addresses %s", dashboardMetricsHelpSuffix),
			dashboardMetricsLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),

		dnsQueryTotal: prometheus.NewDesc(
			prometheus.BuildFQName(dnsMetricsNamespace, "dns_record", "queries_total"),
			fmt.Sprintf("Total number of DNS queries %s", dnsMetricsHelpSuffix),
			dnsMetricsLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		uncachedDNSQueries: prometheus.NewDesc(
			prometheus.BuildFQName(dnsMetricsNamespace, "dns_record", "uncached_queries_total"),
			fmt.Sprintf("Total number of uncached DNS queries %s", dnsMetricsHelpSuffix),
			dnsMetricsLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		staleDNSQueries: prometheus.NewDesc(
			prometheus.BuildFQName(dnsMetricsNamespace, "dns_record", "stale_queries_total"),
			fmt.Sprintf("Total number of DNS queries %s", dnsMetricsHelpSuffix),
			dnsMetricsLabels,
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		componentProcessingTime: prometheus.NewDesc(
			"cloudflare_exporter_component_processing_time_seconds",
			"Component processing time in seconds",
			[]string{"component"},
			prometheus.Labels{"zone_id": zone.ID, "zone_name": zone.Name},
		),
		overallProcessingTime: prometheus.NewDesc(
			"cloudflare_exporter_processing_time_seconds",
			"Processing time in seconds",
			nil,
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

	ch <- e.componentProcessingTime
	ch <- e.overallProcessingTime
}

// Collect fetches the statistics for the configured Cloudflare zone, and
// delivers them as Prometheus metrics. It implements prometheus.Collector.
func (e *ZoneExporter) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	log.Debugf("Getting data for zone %s (%s)", e.zone.Name, e.zone.ID)
	e.collectDashboardAnalytics(ch)
	e.collectDNSAnalytics(ch)
	ch <- prometheus.MustNewConstMetric(e.overallProcessingTime, prometheus.GaugeValue, time.Since(start).Seconds())
}

func (e *ZoneExporter) collectDashboardAnalytics(ch chan<- prometheus.Metric) {
	now := time.Now()
	sinceTime := now.Add(-10080 * time.Minute).UTC() // 7 days
	if e.zone.Plan.LegacyID == "enterprise" {
		sinceTime = now.Add(-30 * time.Minute).UTC() // Anything higher than business gets 1 minute resolution, minimum -30 minutes
	} else if e.zone.Plan.LegacyID == "business" {
		sinceTime = now.Add(-6 * time.Hour).UTC() // Business plans get 15 minute resolution, minimum -6 hours
	} else if e.zone.Plan.LegacyID == "pro" {
		sinceTime = now.Add(-24 * time.Hour).UTC() // Pro plans get 15 minute resolution, minimum -24 hours
	}
	continuous := true
	opts := cloudflare.ZoneAnalyticsOptions{
		Since:      &sinceTime,
		Continuous: &continuous,
	}
	var data []cloudflare.ZoneAnalyticsData
	var err error
	if e.zone.Plan.LegacyID == "enterprise" {
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

		if e.zone.Plan.LegacyID == "enterprise" {
			pop := getPop(entry.ColocationID)
			labels = append(labels, pop.Code, pop.Name, pop.Region)
		}

		latestEntry := entry.Timeseries[len(entry.Timeseries)-1]

		ch <- prometheus.MustNewConstMetric(e.allRequests, prometheus.GaugeValue, float64(latestEntry.Requests.All), labels...)
		ch <- prometheus.MustNewConstMetric(e.cachedRequests, prometheus.GaugeValue, float64(latestEntry.Requests.Cached), labels...)
		ch <- prometheus.MustNewConstMetric(e.uncachedRequests, prometheus.GaugeValue, float64(latestEntry.Requests.Uncached), labels...)
		ch <- prometheus.MustNewConstMetric(e.encryptedRequests, prometheus.GaugeValue, float64(latestEntry.Requests.SSL.Encrypted), labels...)
		ch <- prometheus.MustNewConstMetric(e.unencryptedRequests, prometheus.GaugeValue, float64(latestEntry.Requests.SSL.Unencrypted), labels...)
		for code, count := range latestEntry.Requests.HTTPStatus {
			ch <- prometheus.MustNewConstMetric(e.byStatusRequests, prometheus.GaugeValue, float64(count), append(labels, code)...)
		}
		for contentType, count := range latestEntry.Requests.ContentType {
			ch <- prometheus.MustNewConstMetric(e.byContentTypeRequests, prometheus.GaugeValue, float64(count), append(labels, contentType)...)
		}
		for country, count := range latestEntry.Requests.Country {
			ch <- prometheus.MustNewConstMetric(e.byCountryRequests, prometheus.GaugeValue, float64(count), append(labels, country)...)
		}
		for class, count := range latestEntry.Requests.IPClass {
			ch <- prometheus.MustNewConstMetric(e.byIPClassRequests, prometheus.GaugeValue, float64(count), append(labels, class)...)
		}

		ch <- prometheus.MustNewConstMetric(e.totalBandwidth, prometheus.GaugeValue, float64(latestEntry.Bandwidth.All), labels...)
		ch <- prometheus.MustNewConstMetric(e.cachedBandwidth, prometheus.GaugeValue, float64(latestEntry.Bandwidth.Cached), labels...)
		ch <- prometheus.MustNewConstMetric(e.uncachedBandwidth, prometheus.GaugeValue, float64(latestEntry.Bandwidth.Uncached), labels...)
		ch <- prometheus.MustNewConstMetric(e.encryptedBandwidth, prometheus.GaugeValue, float64(latestEntry.Bandwidth.SSL.Encrypted), labels...)
		ch <- prometheus.MustNewConstMetric(e.unencryptedBandwidth, prometheus.GaugeValue, float64(latestEntry.Bandwidth.SSL.Unencrypted), labels...)
		for contentType, count := range latestEntry.Bandwidth.ContentType {
			ch <- prometheus.MustNewConstMetric(e.byContentTypeBandwidth, prometheus.GaugeValue, float64(count), append(labels, contentType)...)
		}
		for country, count := range latestEntry.Bandwidth.Country {
			ch <- prometheus.MustNewConstMetric(e.byCountryBandwidth, prometheus.GaugeValue, float64(count), append(labels, country)...)
		}

		ch <- prometheus.MustNewConstMetric(e.allThreats, prometheus.GaugeValue, float64(latestEntry.Threats.All), labels...)
		for threatType, count := range latestEntry.Threats.Type {
			ch <- prometheus.MustNewConstMetric(e.byTypeThreats, prometheus.GaugeValue, float64(count), append(labels, threatType)...)
		}
		for country, count := range latestEntry.Threats.Country {
			ch <- prometheus.MustNewConstMetric(e.byCountryThreats, prometheus.GaugeValue, float64(count), append(labels, country)...)
		}

		ch <- prometheus.MustNewConstMetric(e.allPageviews, prometheus.GaugeValue, float64(latestEntry.Pageviews.All), labels...)
		for searchEngine, count := range latestEntry.Pageviews.SearchEngines {
			ch <- prometheus.MustNewConstMetric(e.bySearchEnginePageviews, prometheus.GaugeValue, float64(count), append(labels, searchEngine)...)
		}

		ch <- prometheus.MustNewConstMetric(e.uniqueIPAddresses, prometheus.GaugeValue, float64(latestEntry.Uniques.All), labels...)
	}
	ch <- prometheus.MustNewConstMetric(e.componentProcessingTime, prometheus.GaugeValue, time.Since(now).Seconds(), "dashboard_analytics")
}

func (e *ZoneExporter) collectDNSAnalytics(ch chan<- prometheus.Metric) {
	start := time.Now()
	data, err := e.cf.ZoneDNSAnalyticsByTime(e.zone.ID, cloudflare.ZoneDNSAnalyticsOptions{
		Metrics:    []string{"queryCount", "uncachedCount", "staleCount"},
		Dimensions: e.dnsDimensions,
	})
	if err != nil {
		log.Errorf("failed to get dns analytics from cloudflare for zone %s: %s", e.zone.Name, err)
		return
	}

	for _, row := range data.Rows {
		queryCount := row.Metrics[0][len(row.Metrics[0])-1]
		uncachedCount := row.Metrics[1][len(row.Metrics[1])-1]
		staleCount := row.Metrics[2][len(row.Metrics[2])-1]

		labels := row.Dimensions

		if e.dnsDimensions[len(e.dnsDimensions)-1] == "coloName" {
			labels = row.Dimensions[:len(row.Dimensions)-1]
			pop := getPop(row.Dimensions[len(row.Dimensions)-1])
			labels = append(labels, pop.Code, pop.Name, pop.Region)
		}

		ch <- prometheus.MustNewConstMetric(e.dnsQueryTotal, prometheus.GaugeValue, queryCount, labels...)
		ch <- prometheus.MustNewConstMetric(e.uncachedDNSQueries, prometheus.GaugeValue, uncachedCount, labels...)
		ch <- prometheus.MustNewConstMetric(e.staleDNSQueries, prometheus.GaugeValue, staleCount, labels...)
	}
	ch <- prometheus.MustNewConstMetric(e.componentProcessingTime, prometheus.GaugeValue, time.Since(start).Seconds(), "dns_analytics")
}
