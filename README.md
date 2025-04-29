# OneShotMetricsServer

## Overview

oneShotMetricsServer is a basic web application that exposes one-shot metrics. Metrics are often structured to have one data series but many, many samples. Samples are emitted on every scrape showing the most recent content but not delivering only a single value per event occurrence.

In order to see transitions or events, one can emit a datapoint on every scrape forever until a scrape changes. An example is the timestamp when a pod is created or the current version of some component. This means thousands of data points for one event. This does not draw attention to the event but instead reiterates the state that an event has occurred (the pod is now present, the version is now X).

In order to draw attention to the event itself, we need exactly once delivery to each metrics scraper. We can then easily filter on the data series to see events on the very same timeline and plots where we see changes in other common metrics.

Exactly once delivery means we need to track scrapers so we know when they've seen the events and when we need to continue to hold onto events for them.

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