package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const apiURL = "https://circleci.com/graphql-unstable"
const queryBody = `
query Usage($orgId: String!) {
  plan(orgId: $orgId) {
    billingPeriods(numPeriods: 1) {
      metrics {
        activeUsers {
          totalCount
        }
        projects(filter: {usingDLC: true}) {
          totalCount
        }
        total {
          credits
          seconds
        }
        byProject {
          nodes {
            aggregate {
              credits
              seconds
              dlcCredits
              computeCredits
            }
            project {
              name
            }
          }
        }
      }
    }
  }
}
`
const namespace = "circleci_usage"

var (
	activeUsersGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "active_users",
		Help:      "Active users",
	})
	projectsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "projects",
		Help:      "Projects",
	})
	totalCreditsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "total_credits",
		Help:      "Total credits",
	})
	totalSecondsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "total_seconds",
		Help:      "Total seconds",
	})
	perProjectCreditsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "per_project_credits",
			Help:      "Per project credits",
		},
		[]string{"reponame"},
	)
	perProjectSecondsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "per_project_seconds",
			Help:      "Per project seconds",
		},
		[]string{"reponame"},
	)
	perProjectDLCGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "per_project_dlc_credits",
			Help:      "Per project DLC credits",
		},
		[]string{"reponame"},
	)
	perProjectComputeCreditsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "per_project_compute_credits",
			Help:      "Per project compute credits",
		},
		[]string{"reponame"})
)

type graphqlQuery struct {
	OperationName string            `json:"operationName"`
	Variables     map[string]string `json:"variables"`
	Query         string            `json:"query"`
}

func main() {
	go collectTicker()

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe("0.0.0.0:8000", nil))
}

func collect() {
	query := graphqlQuery{
		OperationName: "Usage",
		Variables: map[string]string{
			"orgId": os.Getenv("CIRCLECI_ORG_ID"),
		},
		Query: queryBody,
	}
	jsonData, err := json.Marshal(query)
	if err != nil {
		panic(err)
	}
	buf := ioutil.NopCloser(bytes.NewBuffer(jsonData))
	url, _ := url.Parse(apiURL)
	req := &http.Request{
		Method: "POST",
		URL:    url,
		Header: http.Header{
			"Authorization": []string{os.Getenv("CIRCLECI_API_TOKEN")},
			"Content-Type":  []string{"application/json"},
		},
		Body: buf,
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	jsonParsed, err := gabs.ParseJSON(body)
	metrics := jsonParsed.S("data").S("plan").S("billingPeriods").Index(0).S("metrics")
	activeUsersGauge.Set(metrics.S("activeUsers").S("totalCount").Data().(float64))
	projectsGauge.Set(metrics.S("projects").S("totalCount").Data().(float64))
	totalCreditsGauge.Set(metrics.S("total").S("credits").Data().(float64))
	totalSecondsGauge.Set(metrics.S("total").S("seconds").Data().(float64))

	projects, projectsErr := metrics.S("byProject").S("nodes").Children()
	if projectsErr != nil {
		panic(projectsErr)
	}

	for _, project := range projects {
		labels := prometheus.Labels{"reponame": project.S("project").S("name").Data().(string)}
		perProjectCreditsGauge.With(labels).Set(project.S("aggregate").S("credits").Data().(float64))
		perProjectSecondsGauge.With(labels).Set(project.S("aggregate").S("seconds").Data().(float64))
		perProjectDLCGauge.With(labels).Set(project.S("aggregate").S("dlcCredits").Data().(float64))
		perProjectComputeCreditsGauge.With(labels).Set(project.S("aggregate").S("computeCredits").Data().(float64))
	}
}

func collectTicker() {
	collected := false

	for {
		collect()

		if !collected {
			prometheus.MustRegister(
				activeUsersGauge,
				projectsGauge,
				totalCreditsGauge,
				totalSecondsGauge,
				perProjectCreditsGauge,
				perProjectSecondsGauge,
				perProjectDLCGauge,
				perProjectComputeCreditsGauge,
			)
		}

		collected = true
		time.Sleep(60 * time.Second)
	}
}
