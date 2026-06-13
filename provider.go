// Package cloudwatch is a Rollops metric-provider plugin backed by AWS
// CloudWatch. It resolves a CloudWatch metric query to a single scalar — the
// latest datapoint of the requested statistic over a lookback window — so
// rollout analysis can gate a canary on CloudWatch metrics.
//
// The query is a small JSON object describing the metric, so the single
// MetricProvider.Query(string) seam carries CloudWatch's multi-field request:
//
//	{"namespace":"AWS/ApplicationELB","metricName":"HTTPCode_Target_5XX_Count",
//	 "stat":"Sum","period":300,
//	 "dimensions":{"LoadBalancer":"app/web/abc","TargetGroup":"targetgroup/web/def"}}
//
// It drives AWS through the `aws` CLI (ambient credentials / IAM role), so no
// AWS SDK or request signing is compiled in.
package cloudwatch

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// Runner runs the aws CLI and returns stdout. Injectable for tests.
type Runner func(ctx context.Context, args ...string) (string, error)

// Provider queries CloudWatch via the aws CLI.
type Provider struct {
	AWS    string // aws binary (default "aws")
	Region string // optional --region override
	Window time.Duration
	Run    Runner
	now    func() time.Time
}

func (p Provider) bin() string {
	if p.AWS != "" {
		return p.AWS
	}
	return "aws"
}

func (p Provider) clock() time.Time {
	if p.now != nil {
		return p.now()
	}
	return time.Now().UTC()
}

func (p Provider) window() time.Duration {
	if p.Window > 0 {
		return p.Window
	}
	return 5 * time.Minute
}

type metricQuery struct {
	Namespace  string            `json:"namespace"`
	MetricName string            `json:"metricName"`
	Stat       string            `json:"stat"`   // Average | Sum | Minimum | Maximum | SampleCount
	Period     int               `json:"period"` // seconds; default 300
	Dimensions map[string]string `json:"dimensions"`
}

type datapoint struct {
	Timestamp time.Time
	Value     float64
}

// Query parses the JSON metric spec, calls get-metric-statistics, and returns
// the most recent datapoint's value for the requested statistic.
func (p Provider) Query(ctx context.Context, query string) (float64, error) {
	if p.Run == nil {
		return 0, fmt.Errorf("cloudwatch: no aws runner configured")
	}
	var q metricQuery
	if err := json.Unmarshal([]byte(query), &q); err != nil {
		return 0, fmt.Errorf("cloudwatch: query must be a JSON metric spec: %w", err)
	}
	if q.Namespace == "" || q.MetricName == "" {
		return 0, fmt.Errorf("cloudwatch: query requires namespace and metricName")
	}
	if q.Stat == "" {
		q.Stat = "Average"
	}
	if q.Period <= 0 {
		q.Period = 300
	}

	now := p.clock()
	args := []string{p.bin(), "cloudwatch", "get-metric-statistics",
		"--namespace", q.Namespace,
		"--metric-name", q.MetricName,
		"--statistics", q.Stat,
		"--period", fmt.Sprintf("%d", q.Period),
		"--start-time", now.Add(-p.window()).Format(time.RFC3339),
		"--end-time", now.Format(time.RFC3339),
		"--output", "json",
	}
	if p.Region != "" {
		args = append(args, "--region", p.Region)
	}
	for name, value := range q.Dimensions {
		args = append(args, "--dimensions", fmt.Sprintf("Name=%s,Value=%s", name, value))
	}

	out, err := p.Run(ctx, args...)
	if err != nil {
		return 0, fmt.Errorf("cloudwatch: get-metric-statistics: %w", err)
	}
	return latestDatapoint(out, q.Stat)
}

// latestDatapoint parses get-metric-statistics JSON and returns the requested
// statistic of the most recent datapoint.
func latestDatapoint(out, stat string) (float64, error) {
	var resp struct {
		Datapoints []map[string]any `json:"Datapoints"`
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		return 0, fmt.Errorf("cloudwatch: decode: %w", err)
	}
	if len(resp.Datapoints) == 0 {
		return 0, fmt.Errorf("cloudwatch: query returned no datapoints")
	}
	points := make([]datapoint, 0, len(resp.Datapoints))
	for _, dp := range resp.Datapoints {
		ts, _ := dp["Timestamp"].(string)
		t, _ := time.Parse(time.RFC3339, ts)
		v, ok := dp[stat].(float64)
		if !ok {
			continue
		}
		points = append(points, datapoint{Timestamp: t, Value: v})
	}
	if len(points) == 0 {
		return 0, fmt.Errorf("cloudwatch: no datapoint carried the %q statistic", stat)
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Timestamp.Before(points[j].Timestamp) })
	return points[len(points)-1].Value, nil
}
