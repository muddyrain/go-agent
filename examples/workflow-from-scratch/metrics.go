package main

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

type requestMetrics struct {
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration prometheus.Histogram
}

func newRequestMetrics() *requestMetrics {
	registry := prometheus.NewRegistry()
	requests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agenthub_proposal_requests_total",
			Help: "Number of POST /tech-proposal responses handled by this server.",
		},
		[]string{"status"},
	)
	duration := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "agenthub_proposal_request_duration_seconds",
		Help: "Handler processing duration of POST /tech-proposal in seconds, including error responses.",
		// 教学阈值：1ms、10ms、100ms、1s；输出bucket是小于等于上界的累计次数。
		Buckets: []float64{0.001, 0.01, 0.1, 1},
	})
	// 固定指标只注册到本服务的新Registry；不使用进程全局注册表。
	registry.MustRegister(requests, duration)
	return &requestMetrics{registry: registry, requests: requests, duration: duration}
}

func (m *requestMetrics) middleware(ctx context.Context, c *app.RequestContext) {
	started := time.Now()
	// 外层计数，内层recovery：Next返回后读取普通响应最终状态，包括恢复后的500。
	c.Next(ctx)
	elapsed := time.Since(started).Seconds()
	if string(c.Method()) != http.MethodPost || c.FullPath() != "/tech-proposal" {
		return
	}
	// 统计Handler处理后的响应，不表示客户端必然收到，也不表示Workflow成功。
	m.requests.WithLabelValues(strconv.Itoa(c.Response.StatusCode())).Inc()
	// 每个匹配请求观察一次；统计下游处理时间，不含网络传输或指标更新本身。
	m.duration.Observe(elapsed)
}

func (m *requestMetrics) handler(_ context.Context, c *app.RequestContext) {
	families, err := m.registry.Gather()
	if err != nil {
		log.Printf("gather request metrics: %v", err)
		c.String(http.StatusInternalServerError, "metrics unavailable\n")
		return
	}

	// 在内存完成全部编码再构造响应；由官方编码器负责HELP/TYPE和标签转义。
	var body bytes.Buffer
	for _, family := range families {
		if _, err := expfmt.MetricFamilyToText(&body, family); err != nil {
			log.Printf("encode request metrics: %v", err)
			c.String(http.StatusInternalServerError, "metrics unavailable\n")
			return
		}
	}
	format := expfmt.NewFormat(expfmt.TypeTextPlain)
	c.Data(http.StatusOK, string(format), body.Bytes())
}
