package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
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

var coloIDRegex = regexp.MustCompile(`(.*) - \((.*)\)`)

type cloudflareOpts struct {
	Key                string
	Email              string
	ZoneName           []string
	DashboardAnalytics bool
	DNSAnalytics       bool
}

// Exporter collects metrics for a Cloudflare zone.
type Exporter struct {
	cf    *cloudflare.API
	Zones []cloudflare.Zone
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

	popStatus     *prometheus.Desc
	serviceStatus *prometheus.Desc
	regionStatus  *prometheus.Desc
	overallStatus *prometheus.Desc
}

// NewExporter returns an initialized exporter.
func NewExporter(opts cloudflareOpts) (*Exporter, error) {
	api, err := cloudflare.New(opts.Key, opts.Email)
	if err != nil {
		log.Fatal(err)
	}

	zones, zonesErr := api.ListZones(opts.ZoneName...)
	if zonesErr != nil {
		return nil, fmt.Errorf("error when listing zones: %s", zonesErr)
	}
	if len(zones) == 0 {
		err := errors.New("couldn't find any zones")
		if opts.ZoneName != nil {
			err = fmt.Errorf("couldn't find any zones named %s", strings.Join(opts.ZoneName, ","))
		}
		return nil, err
	}

	return &Exporter{
		cf:    api,
		Zones: zones,
		colos: map[string]colo{},
		allRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "total"),
			"Total number of requests served",
			[]string{"zone_id", "zone_name"}, nil,
		),
		cachedRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "cached"),
			"Total number of cached requests served",
			[]string{"zone_id", "zone_name"}, nil,
		),
		uncachedRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "uncached"),
			"Total number of requests served from the origin",
			[]string{"zone_id", "zone_name"}, nil,
		),
		encryptedRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "encrypted"),
			"The number of requests served over HTTPS",
			[]string{"zone_id", "zone_name"}, nil,
		),
		unencryptedRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "unencrypted"),
			"The number of requests served over HTTP",
			[]string{"zone_id", "zone_name"}, nil,
		),
		byStatusRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "by_status"),
			"The total number of requests broken out by status code",
			[]string{"zone_id", "zone_name", "status_code"}, nil,
		),
		byContentTypeRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "by_content_type"),
			"The total number of requests broken out by content type",
			[]string{"zone_id", "zone_name", "content_type"}, nil,
		),
		byCountryRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "by_country"),
			"The total number of requests broken out by country",
			[]string{"zone_id", "zone_name", "country_code"}, nil,
		),
		byIPClassRequests: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "requests", "by_ip_class"),
			"The total number of requests broken out by IP class",
			[]string{"zone_id", "zone_name", "ip_class"}, nil,
		),

		totalBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bandwidth", "total_bytes"),
			"The total number of bytes served within the time frame",
			[]string{"zone_id", "zone_name"}, nil,
		),
		cachedBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bandwidth", "cached_bytes"),
			"The total number of bytes that were cached (and served) by Cloudflare",
			[]string{"zone_id", "zone_name"}, nil,
		),
		uncachedBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bandwidth", "uncached_bytes"),
			"The total number of bytes that were fetched and served from the origin server",
			[]string{"zone_id", "zone_name"}, nil,
		),
		encryptedBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bandwidth", "encrypted_bytes"),
			"The total number of bytes served over HTTPS",
			[]string{"zone_id", "zone_name"}, nil,
		),
		unencryptedBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bandwidth", "unencrypted_bytes"),
			"The total number of bytes served over HTTP",
			[]string{"zone_id", "zone_name"}, nil,
		),
		byContentTypeBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bandwidth", "by_content_type_bytes"),
			"The total number of bytes served broken out by content type",
			[]string{"zone_id", "zone_name", "content_type"}, nil,
		),
		byCountryBandwidth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bandwidth", "by_country_bytes"),
			"The total number of bytes served broken out by country",
			[]string{"zone_id", "zone_name", "country_code"}, nil,
		),

		allThreats: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "threats", "total"),
			"The total number of identifiable threats received",
			[]string{"zone_id", "zone_name"}, nil,
		),
		byTypeThreats: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "threats", "by_type"),
			"The total number of identifiable threats received broken out by type",
			[]string{"zone_id", "zone_name", "type"}, nil,
		),
		byCountryThreats: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "threats", "by_country"),
			"The total number of identifiable threats received broken out by country",
			[]string{"zone_id", "zone_name", "country_code"}, nil,
		),

		allPageviews: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pageviews", "total"),
			"The total number of pageviews served",
			[]string{"zone_id", "zone_name"}, nil,
		),
		bySearchEnginePageviews: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pageviews", "by_search_engine"),
			"The total number of pageviews served broken out by search engine",
			[]string{"zone_id", "zone_name", "search_engine"}, nil,
		),

		uniqueIPAddresses: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "unique_ip_addresses", "total"),
			"Total number of unique IP addresses",
			[]string{"zone_id", "zone_name"}, nil,
		),

		dnsQueryTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "dns_record", "queries_total"),
			"Total number of DNS queries",
			[]string{"zone_id", "zone_name", "query_name", "response_code", "origin", "tcp", "ip_version", "colo_id", "query_type", "colo_name", "colo_region"}, nil,
		),
		uncachedDNSQueries: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "dns_record", "uncached_queries_total"),
			"Total number of uncached DNS queries",
			[]string{"zone_id", "zone_name", "query_name", "response_code", "origin", "tcp", "ip_version", "colo_id", "query_type", "colo_name", "colo_region"}, nil,
		),
		staleDNSQueries: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "dns_record", "stale_queries_total"),
			"Total number of DNS queries",
			[]string{"zone_id", "zone_name", "query_name", "response_code", "origin", "tcp", "ip_version", "colo_id", "query_type", "colo_name", "colo_region"}, nil,
		),

		popStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pop", "status"),
			"Cloudflare Point of Presence (PoP) status",
			[]string{"status", "colo_name", "colo_id", "region_name"}, nil,
		),

		regionStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "region", "status"),
			"Cloudflare Region status",
			[]string{"status", "region_name"}, nil,
		),

		serviceStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "service", "status"),
			"Cloudflare service status",
			[]string{"status", "service_name"}, nil,
		),

		overallStatus: prometheus.NewDesc(
			"cloudflare_up",
			"Cloudflare status",
			[]string{"indicator", "description"}, nil,
		),
	}, nil
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
	e.getStatus(ch)
	for _, zone := range e.Zones {
		log.Debugf("Getting data for zone %s (%s)", zone.Name, zone.ID)
		e.getDashboardAnalytics(ch, zone)
		e.getDNSAnalytics(ch, zone)
	}
}

func (e *Exporter) getDashboardAnalytics(ch chan<- prometheus.Metric, z cloudflare.Zone) {
	now := time.Now().UTC()
	sinceTime := now.Add(-10080 * time.Minute) // 7 days
	if z.Plan.Price > 200 {
		sinceTime = now.Add(-30 * time.Minute) // Anything higher than business gets 1 minute resolution, minimum -30 minutes
	} else if z.Plan.Price == 200 {
		sinceTime = now.Add(-6 * time.Hour) // Business plans get 15 minute resolution, minimum -6 hours
	} else if z.Plan.Price == 20 {
		sinceTime = now.Add(-24 * time.Hour) // Pro plans get 15 minute resolution, minimum -24 hours
	}
	continuous := false
	data, err := e.cf.ZoneAnalyticsDashboard(z.ID, cloudflare.ZoneAnalyticsOptions{
		Since:      &sinceTime,
		Until:      &now,
		Continuous: &continuous,
	})
	if err != nil {
		log.Errorf("failed to get dashboard analytics from cloudflare for zone %s: %s", z.Name, err)
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
	for searchEngine, count := range data.Totals.Pageviews.SearchEngines {
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
		log.Errorf("failed to get dns analytics from cloudflare for zone %s: %s", z.Name, err)
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
		coloID := "N/A"
		queryType := "N/A"
		coloName := "N/A"
		coloRegion := "N/A"
		if len(row.Dimensions) >= 6 {
			coloID = row.Dimensions[5]
			coloName = e.colos[coloID].Name
			coloRegion = e.colos[coloID].Region
			if coloID == "SJC-PIG" {
				coloName = e.colos["SJC"].Name
				coloRegion = e.colos["SJC"].Region
			}
		}
		if len(row.Dimensions) == 7 {
			queryType = row.Dimensions[6]
		}

		ch <- prometheus.MustNewConstMetric(e.dnsQueryTotal, prometheus.GaugeValue, queryCount, z.ID, z.Name, queryName, responseCode, origin, tcp, ipVersion, coloID, queryType, coloName, coloRegion)
		ch <- prometheus.MustNewConstMetric(e.uncachedDNSQueries, prometheus.GaugeValue, uncachedCount, z.ID, z.Name, queryName, responseCode, origin, tcp, ipVersion, coloID, queryType, coloName, coloRegion)
		ch <- prometheus.MustNewConstMetric(e.staleDNSQueries, prometheus.GaugeValue, staleCount, z.ID, z.Name, queryName, responseCode, origin, tcp, ipVersion, coloID, queryType, coloName, coloRegion)
	}
}

func getStatusFloat(status string) float64 {
	if status == "none" || status == "operational" {
		return float64(1)
	}
	return float64(0)
}

type colo struct {
	Name   string
	Code   string
	Region string
}

type statusPageSummary struct {
	Page   interface{} `json:"page"`
	Status struct {
		Description string `json:"description"`
		Indicator   string `json:"indicator"`
	} `json:"status"`
	Components []struct {
		Status             string    `json:"status"`
		Name               string    `json:"name"`
		CreatedAt          time.Time `json:"created_at"`
		UpdatedAt          time.Time `json:"updated_at"`
		Position           int       `json:"position"`
		Description        string    `json:"description"`
		Showcase           bool      `json:"showcase"`
		ID                 string    `json:"id"`
		GroupID            string    `json:"group_id"`
		PageID             string    `json:"page_id"`
		Group              bool      `json:"group"`
		OnlyShowIfDegraded bool      `json:"only_show_if_degraded"`
	} `json:"components"`
	Incidents             interface{} `json:"incidents"`
	ScheduledMaintenances interface{} `json:"scheduled_maintenances"`
}

func (e *Exporter) getStatus(ch chan<- prometheus.Metric) {

	req, err := http.NewRequest(http.MethodGet, "https://www.cloudflarestatus.com/api/v2/summary.json", nil)
	if err != nil {
		log.Errorf("failed to get cloudflare status: %s", err)
		return
	}

	req.Header.Set("User-Agent", "go-statuspage")

	res, getErr := http.DefaultClient.Do(req)
	if getErr != nil {
		log.Errorf("failed to get cloudflare status: %s", getErr)
		return
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Errorf("failed to get cloudflare status: %s", readErr)
		return
	}

	statusSummary := statusPageSummary{}
	jsonErr := json.Unmarshal(body, &statusSummary)
	if jsonErr != nil {
		log.Errorln("failed to get cloudflare status", jsonErr)
		return
	}

	groupMap := map[string]string{}

	for _, component := range statusSummary.Components {
		if component.Group {
			groupMap[component.ID] = component.Name
			if !strings.Contains(component.Name, "Cloudflare") {
				ch <- prometheus.MustNewConstMetric(e.regionStatus, prometheus.GaugeValue, getStatusFloat(component.Status), component.Status, component.Name)
			}
		}
	}

	for _, component := range statusSummary.Components {
		if component.Group {
			continue
		}
		matches := coloIDRegex.FindStringSubmatch(component.Name)
		if len(matches) > 0 {
			coloName := matches[1]
			coloCode := matches[2]
			regionName := groupMap[component.GroupID]
			ch <- prometheus.MustNewConstMetric(e.popStatus, prometheus.GaugeValue, getStatusFloat(component.Status), component.Status, coloName, coloCode, regionName)
			e.colos[coloCode] = colo{Name: coloName, Code: coloCode, Region: regionName}
		} else {
			ch <- prometheus.MustNewConstMetric(e.serviceStatus, prometheus.GaugeValue, getStatusFloat(component.Status), component.Status, component.Name)
		}
	}

	ch <- prometheus.MustNewConstMetric(e.overallStatus, prometheus.GaugeValue, getStatusFloat(statusSummary.Status.Indicator), statusSummary.Status.Indicator, statusSummary.Status.Description)
}

func init() {
	prometheus.MustRegister(version.NewCollector("cloudflare_exporter"))
}

func main() {
	var (
		listenAddress = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry $(CLOUDFLARE_EXPORTER_WEB_LISTEN_ADDRESS)").Envar("CLOUDFLARE_EXPORTER_WEB_LISTEN_ADDRESS").Default(":9199").String()
		metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics $(CLOUDFLARE_EXPORTER_WEB_TELEMETRY_PATH)").Envar("CLOUDFLARE_EXPORTER_WEB_TELEMETRY_PATH").Default("/metrics").String()

		opts = cloudflareOpts{}
	)

	kingpin.Flag("cloudflare.api-key", "Cloudflare API key $(CLOUDFLARE_EXPORTER_API_KEY)").Envar("CLOUDFLARE_EXPORTER_API_KEY").Required().StringVar(&opts.Key)
	kingpin.Flag("cloudflare.api-email", "Cloudflare API email $(CLOUDFLARE_EXPORTER_API_EMAIL)").Envar("CLOUDFLARE_EXPORTER_API_EMAIL").Required().StringVar(&opts.Email)
	kingpin.Flag("cloudflare.zone-name", "Zone name(s) to monitor. Provide flag multiple times or comma separated list in environment variable. If not provided, all zones will be monitored. $(CLOUDFLARE_EXPORTER_ZONE_NAME)").Envar("CLOUDFLARE_EXPORTER_ZONE_NAME").StringsVar(&opts.ZoneName)

	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("cloudflare_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infoln("Starting cloudflare_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	// Split CF_ZONE_NAME into slice by comma.
	if len(opts.ZoneName) > 0 {
		if strings.Contains(opts.ZoneName[0], ",") {
			opts.ZoneName = strings.Split(opts.ZoneName[0], ",")
		}
	}

	exporter, err := NewExporter(opts)
	if err != nil {
		log.Fatalln(err)
	}
	prometheus.MustRegister(exporter)

	zoneRows := []string{}
	zoneNames := []string{}
	for _, zone := range exporter.Zones {
		zoneNames = append(zoneNames, zone.Name)
		zoneRows = append(zoneRows, `<tr><td><a target="_blank" href="https://www.cloudflare.com/a/overview/`+zone.Name+`">`+zone.Name+`</a></td><td>`+zone.ID+`</td></tr>`)
	}

	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
                      <head>
                       <title>Cloudflare Exporter</title>
                       <style>table, th, td { border: 1px solid black; text-align: left; }</style>
                      </head>
                      <body>
                        <h1>Cloudflare Exporter</h1>
                        <p><a href='` + *metricsPath + `'>Metrics</a></p>
                        <h2>Config</h2>
                        <h3>Authentication</h3>
                        <p>Authenticated as ` + opts.Email + `</p>
                        <h3>Zones</h3>
                        <table>
                          <thead>
                            <tr>
                              <th>Name</th>
                              <th>ID</th>
                            </tr>
                          </thead>
                          <tbody>` + strings.Join(zoneRows, "\n") + `</tbody>
                        </table>
                        <h2>Build</h2>
                        <pre>` + version.Info() + ` ` + version.BuildContext() + `</pre>
                      </body>
                    </html>`))
	})
	log.Infoln("Starting HTTP server on", *listenAddress)
	log.Infoln("Monitoring zone(s):", strings.Join(zoneNames, ", "))
	log.Fatal(http.ListenAndServe(*listenAddress, nil))

}
