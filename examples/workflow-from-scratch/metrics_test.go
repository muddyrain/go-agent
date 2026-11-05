package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/middlewares/server/recovery"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

// 回读真实响应正文；比较解析出的数值，而不是数文本出现次数。
func readRequestCounters(h *server.Hertz) (map[string]float64, error) {
	r := ut.PerformRequest(h.Engine, "GET", "/metrics", nil).Result()
	if r.StatusCode() != 200 || string(r.Header.Peek("Content-Type")) != string(expfmt.NewFormat(expfmt.TypeTextPlain)) {
		return nil, fmt.Errorf("metrics status=%d type=%q body=%q", r.StatusCode(), r.Header.Peek("Content-Type"), r.Body())
	}
	parser := expfmt.NewTextParser(model.LegacyValidation)
	families, err := parser.TextToMetricFamilies(bytes.NewReader(r.Body()))
	if err != nil {
		return nil, fmt.Errorf("parse request metrics: %w", err)
	}
	values := map[string]float64{}
	// 后续耗时指标是另一个族；计数断言仍只读取Counter族。
	durationFamily := families["agenthub_proposal_request_duration_seconds"]
	expectedFamilies := 1
	if durationFamily != nil {
		expectedFamilies++
		if durationFamily.GetType().String() != "HISTOGRAM" {
			return nil, fmt.Errorf("unexpected duration family: %v", durationFamily)
		}
	}
	family := families["agenthub_proposal_requests_total"]
	if family == nil {
		if len(families) != expectedFamilies-1 {
			return nil, fmt.Errorf("unexpected metric families: %v", families)
		}
		return values, nil
	}
	if len(families) != expectedFamilies || family.GetType().String() != "COUNTER" {
		return nil, fmt.Errorf("unexpected metric families: %v", families)
	}
	for _, metric := range family.Metric {
		if len(metric.Label) != 1 || metric.Label[0].GetName() != "status" || metric.Counter == nil {
			return nil, fmt.Errorf("unexpected counter: %v", metric)
		}
		key := metric.Label[0].GetValue()
		if _, exists := values[key]; exists {
			return nil, fmt.Errorf("duplicate series: %q", key)
		}
		values[key] = metric.GetCounter().GetValue()
	}
	return values, nil
}

func assertRequestCounters(t *testing.T, h *server.Hertz, want map[string]float64) {
	t.Helper()
	got, err := readRequestCounters(h)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("counters=%v want=%v", got, want)
	}
}

func TestRequestMetricsResponsesAndNoSelfCounting(t *testing.T) {
	wf, err := newProposalWorkflow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	h := newProposalServer(wf)
	assertRequestCounters(t, h, map[string]float64{})
	for _, tt := range []struct {
		body, media string
		status      int
	}{
		{`{"task":"Go"}`, "application/json", 200},
		{`{"task":"Second"}`, "application/json", 200},
		{`{"task":"  "}`, "application/json", 400},
		{`{"task":`, "application/json", 400},
		{strings.Repeat("x", 4097), "application/json", 413},
		{`{"task":"Go"}`, "text/plain", 415},
	} {
		r := ut.PerformRequest(h.Engine, "POST", "/tech-proposal?task=ignored", &ut.Body{Body: strings.NewReader(tt.body), Len: len(tt.body)}, ut.Header{Key: "Content-Type", Value: tt.media}).Result()
		if r.StatusCode() != tt.status {
			t.Fatalf("status=%d want=%d", r.StatusCode(), tt.status)
		}
	}
	want := map[string]float64{"200": 2, "400": 2, "413": 1, "415": 1}
	for range 3 {
		assertRequestCounters(t, h, want)
	}
	for _, tt := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/tech-proposal", 405},
		{"POST", "/metrics", 405},
		{"POST", "/unknown/random-path", 404},
	} {
		r := ut.PerformRequest(h.Engine, tt.method, tt.path, nil).Result()
		if r.StatusCode() != tt.status {
			t.Fatalf("method=%s path=%s status=%d want=%d", tt.method, tt.path, r.StatusCode(), tt.status)
		}
	}
	assertRequestCounters(t, h, want)
	other := newProposalServer(wf)
	assertRequestCounters(t, other, map[string]float64{})
}

func TestRequestMetricsSelectThenIncrement(t *testing.T) {
	m := newRequestMetrics()
	h := server.New()
	h.GET("/metrics", m.handler)
	assertRequestCounters(t, h, map[string]float64{})
	counter := m.requests.WithLabelValues("200")
	assertRequestCounters(t, h, map[string]float64{"200": 0})
	counter.Inc()
	counter.Inc()
	m.requests.WithLabelValues("400").Inc()
	assertRequestCounters(t, h, map[string]float64{"200": 2, "400": 1})
	assertRequestCounters(t, h, map[string]float64{"200": 2, "400": 1})
}

func TestRequestMetricsWorkflowErrorAndNextRequest(t *testing.T) {
	wf, err := newProposalWorkflow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	spy := &proposalSpy{Runnable: wf, err: errors.New("private failure")}
	h := newProposalServer(spy)
	var logs bytes.Buffer
	old := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(old)
	body := `{"task":"Go"}`
	r := ut.PerformRequest(h.Engine, "POST", "/tech-proposal", &ut.Body{Body: strings.NewReader(body), Len: len(body)}, ut.Header{Key: "Content-Type", Value: "application/json"}).Result()
	if r.StatusCode() != 500 || string(r.Body()) != "workflow execution failed\n" || !strings.Contains(logs.String(), "private failure") {
		t.Fatalf("status=%d body=%q logs=%q", r.StatusCode(), r.Body(), logs.String())
	}
	assertRequestCounters(t, h, map[string]float64{"500": 1})
	spy.err = nil
	r = ut.PerformRequest(h.Engine, "POST", "/tech-proposal", &ut.Body{Body: strings.NewReader(body), Len: len(body)}, ut.Header{Key: "Content-Type", Value: "application/json"}).Result()
	if r.StatusCode() != 200 {
		t.Fatalf("next request status=%d body=%q", r.StatusCode(), r.Body())
	}
	assertRequestCounters(t, h, map[string]float64{"500": 1, "200": 1})
}

func TestRequestMetricsAfterRecovery(t *testing.T) {
	metrics := newRequestMetrics()
	h := server.New()
	h.Use(metrics.middleware, recovery.Recovery())
	h.POST("/tech-proposal", func(_ context.Context, c *app.RequestContext) {
		c.String(200, "buffered response")
		panic("test panic")
	})
	h.GET("/metrics", metrics.handler)
	r := ut.PerformRequest(h.Engine, "POST", "/tech-proposal", nil).Result()
	if r.StatusCode() != 500 {
		t.Fatalf("status=%d", r.StatusCode())
	}
	assertRequestCounters(t, h, map[string]float64{"500": 1})
}

func TestRequestMetricsConcurrentIncrementAndScrape(t *testing.T) {
	wf, err := newProposalWorkflow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	h := newProposalServer(wf)
	var wg sync.WaitGroup
	for i := range 32 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			body := fmt.Sprintf(`{"task":"task-%d"}`, i)
			r := ut.PerformRequest(h.Engine, "POST", "/tech-proposal", &ut.Body{Body: strings.NewReader(body), Len: len(body)}, ut.Header{Key: "Content-Type", Value: "application/json"}).Result()
			if r.StatusCode() != 200 {
				t.Errorf("status=%d", r.StatusCode())
			}
		}(i)
	}
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := readRequestCounters(h); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	assertRequestCounters(t, h, map[string]float64{"200": 32})
}

// 仅在测试中提供故障Collector，验证Gather错误不泄露诊断。
type brokenMetricsCollector struct{ desc *prometheus.Desc }

func (b brokenMetricsCollector) Describe(ch chan<- *prometheus.Desc) { ch <- b.desc }
func (b brokenMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.NewInvalidMetric(b.desc, errors.New("private collect failure"))
}

func TestRequestMetricsGatherFailure(t *testing.T) {
	m := newRequestMetrics()
	m.registry.MustRegister(brokenMetricsCollector{prometheus.NewDesc("broken_metric", "Test failure", nil, nil)})
	var logs bytes.Buffer
	old := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(old)
	c := app.NewContext(0)
	m.handler(context.Background(), c)
	if c.Response.StatusCode() != 500 || string(c.Response.Body()) != "metrics unavailable\n" || !strings.Contains(logs.String(), "gather request metrics:") || !strings.Contains(logs.String(), "private collect failure") {
		t.Fatalf("status=%d body=%q logs=%q", c.Response.StatusCode(), c.Response.Body(), logs.String())
	}
}
