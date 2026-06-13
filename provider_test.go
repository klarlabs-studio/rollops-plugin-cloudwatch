package cloudwatch

import (
	"context"
	"strings"
	"testing"
	"time"
)

func fixedClock() time.Time { return time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC) }

func TestQuery_ReturnsLatestDatapoint(t *testing.T) {
	var gotArgs []string
	run := func(_ context.Context, args ...string) (string, error) {
		gotArgs = args
		return `{"Datapoints":[
			{"Timestamp":"2026-06-13T11:55:00Z","Sum":3.0},
			{"Timestamp":"2026-06-13T11:59:00Z","Sum":7.0},
			{"Timestamp":"2026-06-13T11:57:00Z","Sum":5.0}
		]}`, nil
	}
	p := Provider{Run: run, now: fixedClock}
	v, err := p.Query(context.Background(), `{"namespace":"AWS/ApplicationELB","metricName":"HTTPCode_Target_5XX_Count","stat":"Sum","period":300,"dimensions":{"LoadBalancer":"app/web/abc"}}`)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if v != 7.0 {
		t.Errorf("value = %v, want 7.0 (latest by timestamp)", v)
	}
	joined := strings.Join(gotArgs, " ")
	for _, want := range []string{"aws cloudwatch get-metric-statistics", "--namespace AWS/ApplicationELB", "--metric-name HTTPCode_Target_5XX_Count", "--statistics Sum", "--period 300", "Name=LoadBalancer,Value=app/web/abc"} {
		if !strings.Contains(joined, want) {
			t.Errorf("aws args missing %q: %v", want, gotArgs)
		}
	}
}

func TestQuery_DefaultsStatAndPeriod(t *testing.T) {
	var gotArgs []string
	run := func(_ context.Context, args ...string) (string, error) {
		gotArgs = args
		return `{"Datapoints":[{"Timestamp":"2026-06-13T11:59:00Z","Average":1.5}]}`, nil
	}
	p := Provider{Run: run, now: fixedClock}
	v, err := p.Query(context.Background(), `{"namespace":"NS","metricName":"M"}`)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if v != 1.5 {
		t.Errorf("value = %v, want 1.5", v)
	}
	joined := strings.Join(gotArgs, " ")
	if !strings.Contains(joined, "--statistics Average") || !strings.Contains(joined, "--period 300") {
		t.Errorf("defaults not applied: %v", gotArgs)
	}
}

func TestQuery_NoDatapoints(t *testing.T) {
	run := func(_ context.Context, _ ...string) (string, error) { return `{"Datapoints":[]}`, nil }
	p := Provider{Run: run, now: fixedClock}
	if _, err := p.Query(context.Background(), `{"namespace":"NS","metricName":"M"}`); err == nil || !strings.Contains(err.Error(), "no datapoints") {
		t.Fatalf("empty datapoints must error, got %v", err)
	}
}

func TestQuery_BadQueryJSON(t *testing.T) {
	p := Provider{Run: func(context.Context, ...string) (string, error) { return "", nil }}
	if _, err := p.Query(context.Background(), "not json"); err == nil {
		t.Fatal("non-JSON query must error")
	}
}

func TestQuery_RequiresNamespaceAndMetric(t *testing.T) {
	p := Provider{Run: func(context.Context, ...string) (string, error) { return "", nil }}
	if _, err := p.Query(context.Background(), `{"namespace":"NS"}`); err == nil {
		t.Fatal("missing metricName must error")
	}
}

func TestQuery_RequiresRunner(t *testing.T) {
	if _, err := (Provider{}).Query(context.Background(), `{"namespace":"NS","metricName":"M"}`); err == nil {
		t.Fatal("missing runner must error")
	}
}
