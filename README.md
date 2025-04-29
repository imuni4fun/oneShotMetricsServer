# OneShotMetricsServer

## Overview

oneShotMetricsServer is a basic web application that exposes one-shot metrics. Metrics are often structured to have one data series but many, many samples. Samples are emitted on every scrape showing the most recent content but not delivering only a single value per event occurrence.

In order to see transitions or events, one can emit a datapoint on every scrape forever until a scrape changes. An example is the timestamp when a pod is created or the current version of some component. This means thousands of data points for one event. This does not draw attention to the event but instead reiterates the state that an event has occurred (the pod is now present, the version is now X).

In order to draw attention to the event itself, we need exactly once delivery to each metrics scraper. We can then easily filter on the data series to see events on the very same timeline and plots where we see changes in other common metrics.

Exactly once delivery means we need to track scrapers so we know when they've seen the events and when we need to continue to hold onto events for them.

Scrapers are tracked by extracting the source IP address for the scrape HTTP request. The port often changes and is not reliable. Currently, based on a small sample set, the IP seems reliable. Headers are observed for instances of request forwarding.

## Approach

oneShotMetricsServer is a lightweight Go web server that serves up a few paths enabling event delivery and metrics scraping. metrics can be delivered universally to this service on the path `/event` via POST with metrics labels delivered via URL params. This can be accessed via a curl or any similar HTTP method.

```
# example
# this posts the current time seconds as an event once per second
# the event will be called events_to_metrics
# the labels will be:
#   - type: test
#   - message: cron_test_manual
#   - seconds: 42
# the value is 1
while true
  do TMP=$(date | cut -d ":" -f 3 | cut -d " " -f 1)
  curl "http://${HOSTNAME}:8090/event?type=test&message=cron_test_manual&seconds=$TMP" -X POST -vv
  sleep 1
done
```

If desired, the value can be set by passing it as if it was a label. Use the label name `value` and format the value as a floating point number.

## Access

oneShotMetricsServer exposes the following paths on the same port.

- GET `/`: returns welcome and `200 OK`
- POST `/event`: receives incoming event with labels as params
- GET `/metrics`: scrapes cached metrics for this scraper
- GET `/healthz`: returns `200 OK`

The default port is 8090 but it can be configured (see "Configuration" section).

## Configuration

Currently, only timestamps are configurable. This is done via an environment variable.

Set `USE_EXPLICIT_TIMESTAMPS` environment variable to `true` (case insensitive) to track timestamps on each event rather than letting the scraper use scrape timestamp for the batch of events. The default is `false`.

If timestamps are used, Prometheus will not mark the data series as stale for a period (default 5m). During this time, the data series will appear as if it was still being emitted. While this service is indeed delivering events exactly once, this Prometheus behavior makes it appear to deliver the value for every scrap within the time period.

If running this Go routine directly, the `config.json` can be updated to use a different port via updating the value at `.server.port`. This may be updated to observe an environment variable as an override in the future.

If running this via the provided Helm charts, visit the [values file](./charts/oneShotMetricsServer/values.yaml) to review the configurable parameters.

## Guidance

Since the intended behavior is exactly once delivery of events, it is recommended to use the default value of `false` for `USE_EXPLICIT_TIMESTAMPS`. Enabling timestamps causes staleness to be bypassed and results in many instances of each event instead of exactly one. The drawback is that the events have the scrape time instead of the actual, more granular, event time.

## Deployment via Helm

```
# add repo
helm repo add oneShotMetricsServer https://imuni4fun.github.io/oneShotMetricsServer

# later, check for updates
helm repo update

# install events-to-metrics
helm install events-to-metrics oneShotMetricsServer/oneShotMetricsServer --set serviceName=events-to-metrics --set serviceNamespace=kaas-events --set netpol.ingress.allowedNamespaceMatchLabels="{kaas-metrics,kaas-monitoring}"

# run test
helm test events-to-metrics
```

## Test

```
 kubectl run --image alpine/curl:8.10.0 test --command -- sh -c 'while true;do TMP=$(date | cut -d ":" -f 3 | cut -d " " -f 1) && curl "http://K8S_SVC.K8S_NS.svc.cluster.local:8090/event?type=test&message=cron_test_manual&seconds=$TMP" -X POST -vv;sleep 1;done'
```