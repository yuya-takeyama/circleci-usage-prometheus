# circleci-usage-prometheus

Prometheus Exporter for CircleCI usage

## Metrics

* `circleci_usage_projects`
* `circleci_usage_active_users`
* `circleci_usage_total_credits`
* `circleci_usage_total_seconds`
* `circleci_usage_per_project_credits`
* `circleci_usage_per_project_seconds`
* `circleci_usage_per_project_dlc_credits`
* `circleci_usage_per_project_compute_credits`

## How it works

It retrieves the data from an undocumented GraphQL endpoint.

It can be unstable by futuree changes.

## Environment Variables

* `CIRCLECI_ORG_ID`: A variable named `orgId` which is passed to GraphQL endpoint
* `CIRCLECI_API_TOKEN`
