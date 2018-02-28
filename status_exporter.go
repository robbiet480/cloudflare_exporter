package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

var popIDRegex = regexp.MustCompile(`(.*) - \((.*)\)`)

// StatusExporter collects metrics about Cloudflare system status.
type StatusExporter struct {
	popStatus     *prometheus.Desc
	serviceStatus *prometheus.Desc
	regionStatus  *prometheus.Desc
	overallStatus *prometheus.Desc
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

func getStatusFloat(status string) float64 {
	if status == "none" || status == "operational" {
		return float64(1)
	}
	return float64(0)
}

// NewStatusExporter returns an initialized StatusExporter.
func NewStatusExporter() *StatusExporter {
	return &StatusExporter{
		popStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pop", "status"),
			"Cloudflare Point of Presence (PoP) status",
			[]string{"status", "pop_name", "pop_id", "region_name"}, nil,
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
	}
}

// Describe describes all the metrics exported by the Cloudflare StatusExporter. It
// implements prometheus.Collector.
func (e *StatusExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.popStatus
	ch <- e.regionStatus
	ch <- e.serviceStatus
	ch <- e.overallStatus
}

// Collect fetches the statistics about Cloudflare system status, and
// delivers them as Prometheus metrics. It implements prometheus.Collector.
func (e *StatusExporter) Collect(ch chan<- prometheus.Metric) {
	req, err := http.NewRequest(http.MethodGet, "https://www.cloudflarestatus.com/api/v2/summary.json", nil)
	if err != nil {
		log.Errorf("failed to get cloudflare status: %s", err)
		return
	}

	req.Header.Set("User-Agent", userAgentHeader)

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
		log.Errorf("failed to get cloudflare status: %s", jsonErr)
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
		matches := popIDRegex.FindStringSubmatch(component.Name)
		if len(matches) > 0 {
			popName := matches[1]
			popCode := matches[2]
			regionName := groupMap[component.GroupID]
			ch <- prometheus.MustNewConstMetric(e.popStatus, prometheus.GaugeValue, getStatusFloat(component.Status), component.Status, popName, popCode, regionName)
			addPop(pop{Name: popName, Code: popCode, Region: regionName})
		} else {
			ch <- prometheus.MustNewConstMetric(e.serviceStatus, prometheus.GaugeValue, getStatusFloat(component.Status), component.Status, component.Name)
		}
	}

	ch <- prometheus.MustNewConstMetric(e.overallStatus, prometheus.GaugeValue, getStatusFloat(statusSummary.Status.Indicator), statusSummary.Status.Indicator, statusSummary.Status.Description)
}
