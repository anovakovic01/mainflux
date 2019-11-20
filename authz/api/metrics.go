package api

import (
	"context"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/mainflux/authz"
)

var _ authz.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     authz.Service
}

// MetricsMiddleware returns given service instance wrappend in the metrics
// middleware.
func MetricsMiddleware(svc authz.Service, counter metrics.Counter, latency metrics.Histogram) authz.Service {
	return metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (mm metricsMiddleware) Authorize(ctx context.Context, p authz.Policy) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "authorize").Add(1)
		mm.latency.With("method", "authorize").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Authorize(ctx, p)
}

func (mm metricsMiddleware) Connect(ctx context.Context, token string, ps map[string]authz.Policy) (map[string]error, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "connect").Add(1)
		mm.latency.With("method", "connect").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Connect(ctx, token, ps)
}

func (mm metricsMiddleware) Disconnect(ctx context.Context, token string, ps map[string]authz.Policy) (map[string]error, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "disconnect").Add(1)
		mm.latency.With("method", "disconnect").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Disconnect(ctx, token, ps)
}

func (mm metricsMiddleware) AddThings(ctx context.Context, owner string, ids ...string) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "add_things").Add(1)
		mm.latency.With("method", "add_things").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.AddThings(ctx, owner, ids...)
}

func (mm metricsMiddleware) AddChannels(ctx context.Context, owner string, ids ...string) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "add_channels").Add(1)
		mm.latency.With("method", "add_channels").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.AddChannels(ctx, owner, ids...)
}

func (mm metricsMiddleware) RemoveThing(ctx context.Context, owner string, id string) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "remove_thing").Add(1)
		mm.latency.With("method", "remove_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.RemoveThing(ctx, owner, id)
}

func (mm metricsMiddleware) RemoveChannel(ctx context.Context, owner string, id string) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "remove_channel").Add(1)
		mm.latency.With("method", "remove_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.RemoveChannel(ctx, owner, id)
}
