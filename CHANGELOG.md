# Changelog

## tip

* FEATURE: add support of the `$__interval` variable in queries. See [this issue](https://github.com/VictoriaMetrics/victorialogs-datasource/issues/61).
  Thanks to @yincongcyincong for [the pull request](https://github.com/VictoriaMetrics/victorialogs-datasource/pull/69).

* BUGFIX: fix issue collecting labels in different log lines. Removed duplicated labels from previous log lines. See [this issue](https://github.com/VictoriaMetrics/victorialogs-datasource/issues/60). 

## v0.4.0

* FEATURE: make retry attempt for datasource requests if returned error is a temporary network error. See [this issue](https://github.com/VictoriaMetrics/victoriametrics-datasource/issues/193)

* BUGFIX: fix dynamic variable issue causing `this is undefined` error. See [this issue](https://github.com/VictoriaMetrics/victorialogs-datasource/issues/64).
* BUGFIX: fix multi-value variable handling to properly format queries. See [this issue](https://github.com/VictoriaMetrics/victorialogs-datasource/issues/65).
* BUGFIX: fix issue with special characters in variable values. See [this issue](https://github.com/VictoriaMetrics/victorialogs-datasource/issues/65).

## v0.3.0

* FEATURE: add beta version of the query builder. The builder allows selecting `field names` and `field value`. See [this issue](https://github.com/VictoriaMetrics/victorialogs-datasource/issues/48).
* FEATURE: add support for loading data for variables, including log field names and values. This feature allows querying `/select/logsql/field_names` for field names and `/select/logsql/field_values` for field values. See [this issue](https://github.com/VictoriaMetrics/victorialogs-datasource/issues/40).

## v0.2.6

* BUGFIX: fix issue with forwarding headers from datasource to the backend or proxy. 
  It might be helpful if a user wants to use some kind of authentication. See [this issue](https://github.com/VictoriaMetrics/victorialogs-datasource/issues/54)

## v0.2.5

* BUGFIX: fix bug with parsing response when time field is empty but message and labels are present.
  It happens when the user tries to show only stats number. See [this issue](https://github.com/VictoriaMetrics/victorialogs-datasource/issues/45).

## v0.2.4

* BUGFIX: fix bug with parsing response when one of the field contains ANSI escape sequences. See [this issue](https://github.com/VictoriaMetrics/victorialogs-datasource/issues/24).

## v0.2.3

* BUGFIX: fix bug with displaying response when one of the stream field is defined and lines are not collected. See [this issue](https://github.com/VictoriaMetrics/victorialogs-datasource/issues/34).

## v0.2.2

* BUGFIX: fix bug with displaying responses with a custom set of fields. See [this issue](https://github.com/VictoriaMetrics/victorialogs-datasource/issues/23).
* BUGFIX: change time range limitation from `_time` in the expression to `start` and `end` query args. See [this issue](https://github.com/VictoriaMetrics/victorialogs-datasource/issues/22).

## v0.2.1

* BUGFIX: change the `metrics` flag from `false` to `true` in `plugin.json` to ensure the plugin appears in the Grafana datasource selection list.

## v0.2.0

* FEATURE: add support for variables in the query. See [this issue](https://github.com/VictoriaMetrics/victorialogs-datasource/issues/5).
* FEATURE: add client-side record limit check for VictoriaLogs < v0.5.0 support. See [this issue](https://github.com/VictoriaMetrics/victorialogs-datasource/issues/9).

* BUGFIX: fix a bug where a manually removed filter would persist in the query after the "Run query" button is clicked.See [this issue](https://github.com/VictoriaMetrics/victorialogs-datasource/issues/8).
* BUGFIX: fix query handling to correctly apply `_time` filter across all queries. See [this issue](https://github.com/VictoriaMetrics/victorialogs-datasource/issues/12) and [this issue](https://github.com/VictoriaMetrics/VictoriaMetrics/issues/5920).
* BUGFIX: fix an issue where sometimes an empty response was returned despite having data in VictoriaLogs. See [this issue](https://github.com/VictoriaMetrics/victorialogs-datasource/issues/10).
