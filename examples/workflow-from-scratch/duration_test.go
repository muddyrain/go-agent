package main

import (
	"bytes"
	"context"
	"errors"
	"log"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/middlewares/server/recovery"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

// 解析HTTP指标正文，独立核对Histogram类型、标签及累计桶。
func readDuration(t *testing.T, h *server.Hertz) *dto.Histogram {
	t.Helper()
	r := ut.PerformRequest(h.Engine, "GET", "/metrics", nil).Result()
	if r.StatusCode() != 200 || string(r.Header.Peek("Content-Type")) != string(expfmt.NewFormat(expfmt.TypeTextPlain)) {
		t.Fatalf("metrics status=%d type=%q", r.StatusCode(), r.Header.Peek("Content-Type"))
	}
	parser := expfmt.NewTextParser(model.LegacyValidation)
	families, err := parser.TextToMetricFamilies(bytes.NewReader(r.Body()))
	if err != nil {
		t.Fatal(err)
	}
	family := families["agenthub_proposal_request_duration_seconds"]
	if family == nil || family.GetType().String() != "HISTOGRAM" || len(family.Metric) != 1 || family.Metric[0].Histogram == nil || len(family.Metric[0].Label) != 0 {
		t.Fatalf("unexpected duration family: %v", family)
	}
	got := family.Metric[0].Histogram
	bounds := []float64{0.001, 0.01, 0.1, 1, math.Inf(1)}
	if len(got.Bucket) != len(bounds) || math.IsNaN(got.GetSampleSum()) || math.IsInf(got.GetSampleSum(), 0) || got.GetSampleSum() < 0 {
		t.Fatalf("unexpected histogram: %v", got)
	}
	var previous uint64
	for i, bucket := range got.Bucket {
		if bucket.GetUpperBound() != bounds[i] || bucket.GetCumulativeCount() < previous || bucket.GetCumulativeCount() > got.GetSampleCount() {
			t.Fatalf("invalid cumulative bucket: %v", bucket)
		}
		previous = bucket.GetCumulativeCount()
	}
	if previous != got.GetSampleCount() {
		t.Fatalf("+Inf bucket count=%d sample count=%d", previous, got.GetSampleCount())
	}
	return got
}

func TestDurationCumulativeBucketsAndRegistration(t *testing.T) {
	m := newRequestMetrics()
	h := server.New()
	h.GET("/metrics", m.handler)
	initial := readDuration(t, h)
	if initial.GetSampleCount() != 0 || initial.GetSampleSum() != 0 {
		t.Fatalf("unexpected initial histogram: %v", initial)
	}
	for _, bucket := range initial.Bucket {
		if bucket.GetCumulativeCount() != 0 {
			t.Fatalf("initial bucket is not zero: %v", bucket)
		}
	}
	// 包含刚好落在0.01边界的样本，以及大于最大有限桶的样本。
	samples := []float64{0.0005, 0.01, 0.2, 1.2}
	var wantSum float64
	for _, sample := range samples {
		m.duration.Observe(sample)
		wantSum += sample
	}
	got := readDuration(t, h)
	counts := []uint64{1, 2, 2, 3, 4}
	if got.GetSampleCount() != uint64(len(samples)) || math.Abs(got.GetSampleSum()-wantSum) > 1e-12 {
		t.Fatalf("histogram=%v want sum=%v", got, wantSum)
	}
	for i, bucket := range got.Bucket {
		if bucket.GetCumulativeCount() != counts[i] {
			t.Fatalf("bucket=%v want count=%d", bucket, counts[i])
		}
	}
	if next := readDuration(t, h); next.GetSampleCount() != got.GetSampleCount() || next.GetSampleSum() != got.GetSampleSum() {
		t.Fatalf("scrape changed histogram: %v", next)
	}
	other := newRequestMetrics()
	otherServer := server.New()
	otherServer.GET("/metrics", other.handler)
	if next := readDuration(t, otherServer); next.GetSampleCount() != 0 || next.GetSampleSum() != 0 {
		t.Fatalf("independent instance shared observations: %v", next)
	}
}

func TestDurationOneObservationCountsSeveralBuckets(t *testing.T) {
	m := newRequestMetrics()
	h := server.New()
	h.GET("/metrics", m.handler)
	m.duration.Observe(0.005)
	got := readDuration(t, h)
	if got.GetSampleCount() != 1 || got.GetSampleSum() != 0.005 {
		t.Fatalf("unexpected observation count or sum: %v", got)
	}
	for i, bucket := range got.Bucket {
		want := uint64(1)
		if i == 0 {
			want = 0
		}
		if bucket.GetCumulativeCount() != want {
			t.Fatalf("bucket=%v want=%d", bucket, want)
		}
	}
}

func TestDurationResponseScopeAndConcurrentCounts(t *testing.T) {
	wf, err := newProposalWorkflow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	h := newProposalServer(wf)
	for _, tt := range []struct {
		body, media string
		status      int
	}{
		{`{"task":"Go"}`, "application/json", 200},
		{`{"task":"Second"}`, "application/json", 200},
		{`{"task":" "}`, "application/json", 400},
		{`{"task":`, "application/json", 400},
		{strings.Repeat("x", 4097), "application/json", 413},
		{`{"task":"Go"}`, "text/plain", 415},
	} {
		r := ut.PerformRequest(h.Engine, "POST", "/tech-proposal?ignored=yes", &ut.Body{Body: strings.NewReader(tt.body), Len: len(tt.body)}, ut.Header{Key: "Content-Type", Value: tt.media}).Result()
		if r.StatusCode() != tt.status {
			t.Fatalf("status=%d want=%d", r.StatusCode(), tt.status)
		}
	}
	got := readDuration(t, h)
	if got.GetSampleCount() != 6 {
		t.Fatalf("histogram=%v", got)
	}
	assertRequestCounters(t, h, map[string]float64{"200": 2, "400": 2, "413": 1, "415": 1})
	for _, tt := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/tech-proposal", 405},
		{"POST", "/metrics", 405},
		{"POST", "/missing", 404},
	} {
		r := ut.PerformRequest(h.Engine, tt.method, tt.path, nil).Result()
		if r.StatusCode() != tt.status {
			t.Fatalf("method=%s path=%s status=%d", tt.method, tt.path, r.StatusCode())
		}
	}
	if next := readDuration(t, h); next.GetSampleCount() != 6 || next.GetSampleSum() != got.GetSampleSum() {
		t.Fatalf("excluded requests observed: %v", next)
	}
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := `{"task":"Go"}`
			r := ut.PerformRequest(h.Engine, "POST", "/tech-proposal", &ut.Body{Body: strings.NewReader(body), Len: len(body)}, ut.Header{Key: "Content-Type", Value: "application/json"}).Result()
			if r.StatusCode() != 200 {
				t.Errorf("concurrent status=%d", r.StatusCode())
			}
		}()
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
	if next := readDuration(t, h); next.GetSampleCount() != 38 {
		t.Fatalf("count=%d", next.GetSampleCount())
	}
	assertRequestCounters(t, h, map[string]float64{"200": 34, "400": 2, "413": 1, "415": 1})
}

func TestDurationRecoveryAndWorkflowError(t *testing.T) {
	m := newRequestMetrics()
	h := server.New()
	h.Use(m.middleware, recovery.Recovery())
	h.GET("/metrics", m.handler)
	h.POST("/tech-proposal", func(_ context.Context, c *app.RequestContext) {
		c.String(200, "buffered")
		panic("duration panic")
	})
	if r := ut.PerformRequest(h.Engine, "POST", "/tech-proposal", nil).Result(); r.StatusCode() != 500 {
		t.Fatalf("recovered status=%d", r.StatusCode())
	}
	if got := readDuration(t, h); got.GetSampleCount() != 1 {
		t.Fatalf("recovered panic not observed: %v", got)
	}
	assertRequestCounters(t, h, map[string]float64{"500": 1})

	wf, err := newProposalWorkflow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	spy := &proposalSpy{Runnable: wf, err: errors.New("private failure")}
	other := newProposalServer(spy)
	var logs bytes.Buffer
	old := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(old)
	body := `{"task":"Go"}`
	r := ut.PerformRequest(other.Engine, "POST", "/tech-proposal", &ut.Body{Body: strings.NewReader(body), Len: len(body)}, ut.Header{Key: "Content-Type", Value: "application/json"}).Result()
	if r.StatusCode() != 500 || string(r.Body()) != "workflow execution failed\n" {
		t.Fatalf("workflow failure status=%d body=%q", r.StatusCode(), r.Body())
	}
	if got := readDuration(t, other); got.GetSampleCount() != 1 {
		t.Fatalf("workflow error not observed: %v", got)
	}
	spy.err = nil
	if next := ut.PerformRequest(other.Engine, "POST", "/tech-proposal", &ut.Body{Body: strings.NewReader(body), Len: len(body)}, ut.Header{Key: "Content-Type", Value: "application/json"}).Result(); next.StatusCode() != 200 {
		t.Fatalf("next request status=%d", next.StatusCode())
	}
	if got := readDuration(t, other); got.GetSampleCount() != 2 {
		t.Fatalf("next request not observed: %v", got)
	}
	assertRequestCounters(t, other, map[string]float64{"500": 1, "200": 1})
}

func TestDurationMeasuresDownstreamInSeconds(t *testing.T) {
	m := newRequestMetrics()
	h := server.New()
	h.Use(m.middleware, recovery.Recovery())
	h.GET("/metrics", m.handler)
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	h.POST("/tech-proposal", func(_ context.Context, c *app.RequestContext) {
		close(entered)
		<-release
		c.String(200, "ok")
	})
	var once sync.Once
	t.Cleanup(func() {
		once.Do(func() { close(release) })
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("request goroutine did not exit")
		}
	})
	testStarted := time.Now()
	go func() { defer close(done); ut.PerformRequest(h.Engine, "POST", "/tech-proposal", nil) }()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("handler did not enter")
	}
	if got := readDuration(t, h); got.GetSampleCount() != 0 {
		t.Fatalf("observed before handler returned: %v", got)
	}
	// 仅测试夹具等待，不给业务实现增加Sleep；以外层实测时间作为宽松上限。
	time.Sleep(20 * time.Millisecond)
	once.Do(func() { close(release) })
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("handler did not finish")
	}
	upperSeconds := time.Since(testStarted).Seconds() + 0.1
	got := readDuration(t, h)
	if got.GetSampleCount() != 1 || got.GetSampleSum() < 0.02 || got.GetSampleSum() > upperSeconds {
		t.Fatalf("expected downstream duration in seconds, upper=%v: %v", upperSeconds, got)
	}
}
