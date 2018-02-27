# Cloudflare Exporter [![Build Status](https://travis-ci.org/robbiet480/cloudflare_exporter.svg)][travis]

[![Docker Repository on Quay](https://quay.io/repository/robbiet480/cloudflare_exporter/status)][quay]
[![Docker Pulls](https://img.shields.io/docker/pulls/robbiet480/cloudflare_exporter.svg?maxAge=604800)][hub]

Export Cloudflare zone and DNS analytics to Prometheus.

Exported metrics and time precision depends on your plan.

To run it:

```bash
make
./cloudflare_exporter [flags]
```

## Exported Metrics

| Metric | Meaning | Labels |
| ------ | ------- | ------ |
| cloudflare_exporter_build_info | A metric with a constant '1' value labeled by version, revision, branch, and goversion from which cloudflare_exporter was built. | `version`, `revision`, `branch`, `goversion` |
| cloudflare_bandwidth_by_content_type_bytes | The total number of bytes served broken out by content type | `zone_id`, `zone_name`, `content_type` |
| cloudflare_bandwidth_by_country_bytes | The total number of bytes served broken out by country | `zone_id`, `zone_name`, `country_code` |
| cloudflare_bandwidth_cached_bytes | The total number of bytes that were cached (and served) by Cloudflare | `zone_id`, `zone_name` |
| cloudflare_bandwidth_encrypted_bytes | The total number of bytes served over HTTPS | `zone_id`, `zone_name` |
| cloudflare_bandwidth_total_bytes | The total number of bytes served within the time frame | `zone_id`, `zone_name` |
| cloudflare_bandwidth_uncached_bytes | The total number of bytes that were fetched and served from the origin server | `zone_id`, `zone_name` |
| cloudflare_bandwidth_unencrypted_bytes | The total number of bytes served over HTTP | `zone_id`, `zone_name` |
| cloudflare_dns_record_queries_total | Total number of DNS queries | `zone_id`, `zone_name`, `query_name`, `response_code`, `origin`, `tcp`, `ip_version`, `colo_id`, `colo_name`, `colo_region`, `query_type` |
| cloudflare_dns_record_stale_queries_total | Total number of DNS queries | `zone_id`, `zone_name`, `query_name`, `response_code`, `origin`, `tcp`, `ip_version`, `colo_id`, `colo_name`, `colo_region`, `query_type` |
| cloudflare_dns_record_uncached_queries_total | Total number of uncached DNS queries | `zone_id`, `zone_name`, `query_name`, `response_code`, `origin`, `tcp`, `ip_version`, `colo_id`, `colo_name`, `colo_region`, `query_type` |
| cloudflare_pageviews_by_search_engine | The total number of pageviews served broken out by search engine | `zone_id`, `zone_name`, `search_engine` |
| cloudflare_pageviews_total | The total number of pageviews served | `zone_id`, `zone_name` |
| cloudflare_pop_status | Cloudflare Point of Presence (PoP) status | `status`, `colo_name`, `colo_id`, `region_name` |
| cloudflare_region_status | Cloudflare Region status | `status`, `region_name` |
| cloudflare_requests_by_content_type | The total number of requests broken out by content type | `zone_id`, `zone_name`, `content_type` |
| cloudflare_requests_by_country | The total number of requests broken out by country | `zone_id`, `zone_name`, `country_code` |
| cloudflare_requests_by_ip_class | The total number of requests broken out by IP class | `zone_id`, `zone_name`, `ip_class` |
| cloudflare_requests_by_status | The total number of requests broken out by status code | `zone_id`, `zone_name`, `status_code` |
| cloudflare_requests_cached | Total number of cached requests served | `zone_id`, `zone_name` |
| cloudflare_requests_encrypted | The number of requests served over HTTPS | `zone_id`, `zone_name` |
| cloudflare_requests_total | Total number of requests served | `zone_id`, `zone_name` |
| cloudflare_requests_uncached | Total number of requests served from the origin | `zone_id`, `zone_name` |
| cloudflare_requests_unencrypted | The number of requests served over HTTP | `zone_id`, `zone_name` |
| cloudflare_service_status | Cloudflare service status | `status`, `service_name` |
| cloudflare_threats_by_country | The total number of identifiable threats received broken out by country | `zone_id`, `zone_name`, `country_code` |
| cloudflare_threats_by_type | The total number of identifiable threats received broken out by type | `zone_id`, `zone_name`, `type` |
| cloudflare_threats_total | The total number of identifiable threats received | `zone_id`, `zone_name` |
| cloudflare_unique_ip_addresses_total | Total number of unique IP addresses | `zone_id`, `zone_name` |
| cloudflare_up | Cloudflare status | `indicator`, `description` |

### Configuration

```bash
./cloudflare_exporter --help
```

| Name | Description | Optional | Default | Flag | Environment Variable |
|--------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|------------|------------------------|-----------------------------------------|
| API Key | Your Cloudflare API key | Required | N/A | --cloudflare.api-key | CLOUDFLARE_EXPORTER_API_KEY |
| API Email | Your Cloudflare API email | Required | N/A | --cloudflare.api-email | CLOUDFLARE_EXPORTER_API_EMAIL |
| Zone Name(s) | Cloudflare zone name(s) to monitor. Provide flag multiple times or comma separated list in environment variable. If not provided, all zones will be monitored. | Optional | all zones | --cloudflare.zone-name |  CLOUDFLARE_EXPORTER_ZONE_NAME |
| Web Listen Address | Address to listen on for web interface and telemetry | Required | `:9199` | --web.listen-address | CLOUDFLARE_EXPORTER_WEB_LISTEN_ADDRESS |
| Web Telemetry Path | Path under which to expose metrics | Required | `/metrics` | --web.telemetry-path |  CLOUDFLARE_EXPORTER_WEB_TELEMETRY_PATH |

## Using Docker

You can deploy this exporter using the [robbiet480/cloudflare_exporter](https://registry.hub.docker.com/u/robbiet480/cloudflare_exporter/) Docker image.

For example:

```bash
docker pull robbiet480/cloudflare_exporter

docker run -d -p 9199:9199 robbiet480/cloudflare_exporter --cloudflare.api-key=myapikey --cloudflare.api-email=me@domain.name
```

# LICENSE

Apache-2.0

[hub]: https://hub.docker.com/r/robbiet480/cloudflare_exporter/
[travis]: https://travis-ci.org/robbiet480/cloudflare_exporter
[quay]: https://quay.io/repository/robbiet480/cloudflare_exporter
