package eventbus

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const instrumentScope = "eventbus"

// MetricsMiddleware 为 Dispatcher 添加以下指标埋点：
//
//   - eventbus_dispatch_total (Counter)     — 分发调用次数
//   - eventbus_handler_duration_seconds (Histogram) — Handler 执行耗时（秒）
//
// 通过全局 MeterProvider 获取 Meter，未设置 Provider 时为零开销。
func MetricsMiddleware() Middleware {
	meter := otel.Meter(instrumentScope)

	dispatchCounter, _ := meter.Int64Counter(
		"eventbus_dispatch_total",
		metric.WithDescription("事件分发总次数"),
	)
	handlerDuration, _ := meter.Float64Histogram(
		"eventbus_handler_duration_seconds",
		metric.WithDescription("事件处理器耗时（秒）"),
	)

	return func(next Dispatcher) Dispatcher {
		return DispatchFunc(func(event Event, handlers []Handler) {
			ctx := context.Background()

			dispatchCounter.Add(ctx, 1,
				metric.WithAttributes(
					attribute.String("event_id", event.Id),
				),
			)

			// 包装每个 handler，在 defer 中记录耗时。
			// defer 在 panic 时也会执行，因此耗时始终会被记录。
			instrumented := make([]Handler, len(handlers))
			for i, h := range handlers {
				h := h
				instrumented[i] = func(e Event) {
					start := time.Now()
					defer func() {
						handlerDuration.Record(ctx, time.Since(start).Seconds(),
							metric.WithAttributes(
								attribute.String("event_id", event.Id),
							),
						)
					}()
					h(e)
				}
			}

			next.Dispatch(event, instrumented)
		})
	}
}

// TracingMiddleware 为每次事件分发创建一个 Span：
//
//	eventbus.dispatch
//	  └── eventbus.handler
//	      └── eventbus.handler
//
// 通过全局 TracerProvider 获取 Tracer，未设置 Provider 时为零开销。
func TracingMiddleware() Middleware {
	tracer := otel.Tracer(instrumentScope)

	return func(next Dispatcher) Dispatcher {
		return DispatchFunc(func(event Event, handlers []Handler) {
			ctx, span := tracer.Start(context.Background(), "eventbus.dispatch",
				trace.WithAttributes(
					attribute.String("event_id", event.Id),
				),
			)
			defer span.End()

			// 为每个 handler 创建子 Span
			instrumented := make([]Handler, len(handlers))
			for i, h := range handlers {
				h := h
				instrumented[i] = func(e Event) {
					_, handlerSpan := tracer.Start(ctx, "eventbus.handler",
						trace.WithAttributes(
							attribute.String("event_id", event.Id),
						),
					)
					defer handlerSpan.End()
					h(e)
				}
			}

			next.Dispatch(event, instrumented)
		})
	}
}
