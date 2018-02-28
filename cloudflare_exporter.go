package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

const (
	namespace = "cloudflare"
)

type cloudflareOpts struct {
	Key                string
	Email              string
	ZoneName           []string
	DashboardAnalytics bool
	DNSAnalytics       bool
}

var reg = prometheus.NewPedanticRegistry()

func init() {
	reg.MustRegister(version.NewCollector("cloudflare_exporter"))
	initPops()
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

	// Split CLOUDFLARE_EXPORTER_ZONE_NAME into slice by comma.
	if len(opts.ZoneName) > 0 {
		if strings.Contains(opts.ZoneName[0], ",") {
			opts.ZoneName = strings.Split(opts.ZoneName[0], ",")
		}
	}

	api, err := cloudflare.New(opts.Key, opts.Email)
	if err != nil {
		log.Fatal(err)
	}

	zones, zonesErr := api.ListZones(opts.ZoneName...)
	if zonesErr != nil {
		log.Fatalf("error when listing zones: %s", zonesErr)
	}
	if len(zones) == 0 {
		err := errors.New("couldn't find any zones")
		if opts.ZoneName != nil {
			err = fmt.Errorf("couldn't find any zones named %s", strings.Join(opts.ZoneName, ","))
		}
		log.Fatal(err)
	}

	zoneRows := []string{}
	zoneNames := []string{}
	reg.MustRegister(NewStatusExporter())
	for _, zone := range zones {
		reg.MustRegister(NewZoneExporter(api, zone))
		zoneNames = append(zoneNames, zone.Name)
		zoneRows = append(zoneRows, `<tr><td><a target="_blank" href="https://www.cloudflare.com/a/overview/`+zone.Name+`">`+zone.Name+`</a></td><td>`+zone.ID+`</td></tr>`)
	}

	http.Handle(*metricsPath, promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	http.HandleFunc("/pops.json", func(w http.ResponseWriter, r *http.Request) {
		marshalledPoPs, _ := json.Marshal(pops)
		w.Header().Set("Content-Type", "application/json")
		w.Write(marshalledPoPs)
	})
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
                        <h2>Misc</h2>
                        <p><a href="/pops.json">Here's all the Points of Presence (PoPs) I know about</a></p>
                        <h2>Build</h2>
                        <pre>` + version.Info() + ` ` + version.BuildContext() + `</pre>
                      </body>
                    </html>`))
	})
	log.Infoln("Starting HTTP server on", *listenAddress)
	log.Infoln("Exposing metrics for zone(s):", strings.Join(zoneNames, ", "))
	log.Fatal(http.ListenAndServe(*listenAddress, nil))

}
