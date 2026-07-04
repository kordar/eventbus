package eventbus_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kordar/eventbus"
)

func TestEventBus_Emit(t *testing.T) {
	bus := eventbus.NewEventBus()

	called := make(chan struct{}, 1)
	bus.On("post", func(event eventbus.Event) {
		called <- struct{}{}
	})

	bus.Emit("post", eventbus.Event{Payload: "payload"})

	select {
	case <-called:
	case <-time.After(2 * time.Second):
		t.Fatalf("handler not called")
	}
}

func TestEventBus_Emit_GoroutineDispatcher(t *testing.T) {
	bus := eventbus.NewEventBus(
		eventbus.WithDispatchers(map[string]eventbus.Dispatcher{
			"async": eventbus.GoroutineDispatcher{},
		}),
	)

	called := make(chan struct{}, 1)
	bus.On("t", func(event eventbus.Event) {
		panic("boom")
	})
	bus.On("t", func(event eventbus.Event) {
		called <- struct{}{}
	})

	bus.Emit("t", eventbus.Event{Dispatcher: "async"})

	select {
	case <-called:
	case <-time.After(2 * time.Second):
		t.Fatalf("handler not called")
	}
}

func TestEventBus_Off(t *testing.T) {
	bus := eventbus.NewEventBus()

	called := make(chan struct{}, 1)
	bus.On("t", func(event eventbus.Event) {
		called <- struct{}{}
	})
	bus.Off("t")
	bus.Emit("t", eventbus.Event{})

	select {
	case <-called:
		t.Fatal("handler should not be called after Off")
	case <-time.After(100 * time.Millisecond):
		// expected
	}
}

func TestEventBus_Off_OnlyTargetTopic(t *testing.T) {
	bus := eventbus.NewEventBus()

	called := make(chan struct{}, 1)
	bus.On("a", func(event eventbus.Event) {
		called <- struct{}{}
	})

	bus.Off("b")
	bus.Emit("a", eventbus.Event{})

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("handler should still be called")
	}
}

func TestEventBus_OnNil(t *testing.T) {
	bus := eventbus.NewEventBus()
	bus.On("t", nil)
	bus.Emit("t", eventbus.Event{})
}

// ---------- Dispatcher 接口测试 ----------

func TestEventBus_WithDispatcher_Custom(t *testing.T) {
	var count atomic.Int64
	cd := eventbus.DispatchFunc(func(event eventbus.Event, handlers []eventbus.Handler) {
		count.Add(1)
		for _, h := range handlers {
			h(event)
		}
	})

	bus := eventbus.NewEventBus(eventbus.WithDispatcher(cd))

	called := make(chan struct{}, 1)
	bus.On("t", func(event eventbus.Event) {
		called <- struct{}{}
	})

	bus.Emit("t", eventbus.Event{})

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("handler not called")
	}

	if c := count.Load(); c != 1 {
		t.Fatalf("expected 1 dispatch, got %d", c)
	}
}

func TestEventBus_WithSelector(t *testing.T) {
	var syncCount, asyncCount atomic.Int64

	bus := eventbus.NewEventBus(
		eventbus.WithSelector(func(topic string, event eventbus.Event) eventbus.Dispatcher {
			if event.Id == "fast" {
				return eventbus.SyncDispatcher{}
			}
			return eventbus.GoroutineDispatcher{}
		}),
		eventbus.WithMiddleware(func(next eventbus.Dispatcher) eventbus.Dispatcher {
			return eventbus.DispatchFunc(func(event eventbus.Event, handlers []eventbus.Handler) {
				if event.Id == "fast" {
					syncCount.Add(1)
				} else {
					asyncCount.Add(1)
				}
				next.Dispatch(event, handlers)
			})
		}),
	)

	called := make(chan struct{}, 2)
	bus.On("t", func(event eventbus.Event) {
		called <- struct{}{}
	})

	bus.Emit("t", eventbus.Event{Id: "fast"})
	bus.Emit("t", eventbus.Event{Id: "slow"})

	for i := 0; i < 2; i++ {
		select {
		case <-called:
		case <-time.After(time.Second):
			t.Fatal("handler not called")
		}
	}

	if syncCount.Load() != 1 || asyncCount.Load() != 1 {
		t.Fatalf("expected 1 sync + 1 async, got sync=%d async=%d", syncCount.Load(), asyncCount.Load())
	}
}

func TestEventBus_Middleware_Logging(t *testing.T) {
	var mwCalled atomic.Int64

	bus := eventbus.NewEventBus(
		eventbus.WithMiddleware(func(next eventbus.Dispatcher) eventbus.Dispatcher {
			return eventbus.DispatchFunc(func(event eventbus.Event, handlers []eventbus.Handler) {
				mwCalled.Add(1)
				next.Dispatch(event, handlers)
			})
		}),
	)

	called := make(chan struct{}, 1)
	bus.On("t", func(event eventbus.Event) {
		called <- struct{}{}
	})

	bus.Emit("t", eventbus.Event{})

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("handler not called")
	}

	if mwCalled.Load() != 1 {
		t.Fatalf("expected middleware called once, got %d", mwCalled.Load())
	}
}

func TestEventBus_DispatchFunc_Adapter(t *testing.T) {
	var dispatched atomic.Int64
	df := eventbus.DispatchFunc(func(event eventbus.Event, handlers []eventbus.Handler) {
		dispatched.Add(int64(len(handlers)))
		for _, h := range handlers {
			h(event)
		}
	})

	bus := eventbus.NewEventBus(eventbus.WithDispatcher(df))

	called := make(chan struct{}, 1)
	bus.On("t", func(event eventbus.Event) {
		called <- struct{}{}
	})

	bus.Emit("t", eventbus.Event{})

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("handler not called")
	}

	if dispatched.Load() != 1 {
		t.Fatalf("expected dispatched 1 handler, got %d", dispatched.Load())
	}
}

func TestEventBus_RecoveryMiddleware(t *testing.T) {
	bus := eventbus.NewEventBus(
		eventbus.WithDispatcher(eventbus.DispatchFunc(func(event eventbus.Event, handlers []eventbus.Handler) {
			panic("dispatcher panic")
		})),
		eventbus.WithMiddleware(eventbus.RecoveryMiddleware()),
	)

	bus.On("t", func(event eventbus.Event) {})

	bus.Emit("t", eventbus.Event{})
	t.Log("OK: recovery middleware caught dispatcher panic")
}

func TestEventBus_NoHandlers_DispatchersNotCalled(t *testing.T) {
	var dispatched atomic.Int64
	bus := eventbus.NewEventBus(
		eventbus.WithDispatcher(eventbus.DispatchFunc(func(event eventbus.Event, handlers []eventbus.Handler) {
			dispatched.Add(1)
		})),
	)

	bus.Emit("t", eventbus.Event{})

	if dispatched.Load() != 0 {
		t.Fatalf("dispatcher should not be called when no handlers registered")
	}
}

func TestEventBus_SyncDispatcher_ConcurrentSafety(t *testing.T) {
	bus := eventbus.NewEventBus()
	var wg sync.WaitGroup

	const goroutines = 50
	const emits = 1_000
	var total atomic.Int64

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < emits; i++ {
				bus.On("t", func(event eventbus.Event) {
					total.Add(1)
				})
				bus.Emit("t", eventbus.Event{})
				bus.Off("t")
			}
		}()
	}
	wg.Wait()

	t.Logf("total handled: %d", total.Load())
}

// ---------- 内置 Dispatcher 的直接测试 ----------

func TestSyncDispatcher(t *testing.T) {
	d := eventbus.SyncDispatcher{}
	var called bool
	d.Dispatch(eventbus.Event{}, []eventbus.Handler{
		func(event eventbus.Event) { called = true },
	})
	if !called {
		t.Fatal("SyncDispatcher should call handler")
	}
}

func TestGoroutineDispatcher(t *testing.T) {
	d := eventbus.GoroutineDispatcher{}
	done := make(chan struct{}, 1)
	d.Dispatch(eventbus.Event{}, []eventbus.Handler{
		func(event eventbus.Event) { done <- struct{}{} },
	})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("GoroutineDispatcher should call handler")
	}
}

// ---------- 命名分发器测试 ----------

func TestNamedDispatcher_WithDispatchers(t *testing.T) {
	var syncHit, asyncHit atomic.Int64

	bus := eventbus.NewEventBus(
		eventbus.WithDispatchers(map[string]eventbus.Dispatcher{
			"sync": eventbus.DispatchFunc(func(e eventbus.Event, hs []eventbus.Handler) {
				syncHit.Add(1)
				for _, h := range hs {
					h(e)
				}
			}),
			"async": eventbus.DispatchFunc(func(e eventbus.Event, hs []eventbus.Handler) {
				asyncHit.Add(1)
				for _, h := range hs {
					h(e)
				}
			}),
		}),
	)

	called := make(chan struct{}, 2)
	bus.On("t", func(event eventbus.Event) { called <- struct{}{} })

	bus.Emit("t", eventbus.Event{Dispatcher: "sync"})
	bus.Emit("t", eventbus.Event{Dispatcher: "async"})

	for i := 0; i < 2; i++ {
		select {
		case <-called:
		case <-time.After(time.Second):
			t.Fatal("handler not called")
		}
	}

	if syncHit.Load() != 1 || asyncHit.Load() != 1 {
		t.Fatalf("expected sync=1 async=1, got sync=%d async=%d", syncHit.Load(), asyncHit.Load())
	}
}

func TestNamedDispatcher_RegisterDispatcher(t *testing.T) {
	bus := eventbus.NewEventBus()

	var called atomic.Int64
	bus.RegisterDispatcher("custom", eventbus.DispatchFunc(func(e eventbus.Event, hs []eventbus.Handler) {
		called.Add(1)
		for _, h := range hs {
			h(e)
		}
	}))

	done := make(chan struct{}, 1)
	bus.On("t", func(event eventbus.Event) { done <- struct{}{} })

	bus.Emit("t", eventbus.Event{Dispatcher: "custom"})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler not called")
	}

	if called.Load() != 1 {
		t.Fatalf("expected dispatcher called once, got %d", called.Load())
	}
}

func TestNamedDispatcher_Unregister(t *testing.T) {
	bus := eventbus.NewEventBus()

	bus.RegisterDispatcher("slow", eventbus.DispatchFunc(func(e eventbus.Event, hs []eventbus.Handler) {
		for _, h := range hs {
			h(e)
		}
	}))
	bus.UnregisterDispatcher("slow")

	var called atomic.Int64
	bus.On("t", func(event eventbus.Event) { called.Add(1) })

	bus.Emit("t", eventbus.Event{Dispatcher: "slow"})

	if called.Load() != 1 {
		t.Fatalf("expected fallback to default dispatcher, got %d", called.Load())
	}
}

func TestNamedDispatcher_CoexistWithDefault(t *testing.T) {
	var defaultHit, namedHit atomic.Int64

	bus := eventbus.NewEventBus(
		eventbus.WithDispatcher(eventbus.DispatchFunc(func(e eventbus.Event, hs []eventbus.Handler) {
			defaultHit.Add(1)
			for _, h := range hs {
				h(e)
			}
		})),
		eventbus.WithDispatchers(map[string]eventbus.Dispatcher{
			"batch": eventbus.DispatchFunc(func(e eventbus.Event, hs []eventbus.Handler) {
				namedHit.Add(1)
				for _, h := range hs {
					h(e)
				}
			}),
		}),
	)

	called := make(chan struct{}, 2)
	bus.On("t", func(event eventbus.Event) { called <- struct{}{} })

	bus.Emit("t", eventbus.Event{})
	bus.Emit("t", eventbus.Event{Dispatcher: "batch"})

	for i := 0; i < 2; i++ {
		select {
		case <-called:
		case <-time.After(time.Second):
			t.Fatal("handler not called")
		}
	}

	if defaultHit.Load() != 1 || namedHit.Load() != 1 {
		t.Fatalf("expected default=1 named=1, got default=%d named=%d", defaultHit.Load(), namedHit.Load())
	}
}

func TestNamedDispatcher_PriorityOverSelector(t *testing.T) {
	var selectorHit, namedHit atomic.Int64

	bus := eventbus.NewEventBus(
		eventbus.WithSelector(func(topic string, e eventbus.Event) eventbus.Dispatcher {
			return eventbus.DispatchFunc(func(ev eventbus.Event, hs []eventbus.Handler) {
				selectorHit.Add(1)
				for _, h := range hs {
					h(ev)
				}
			})
		}),
		eventbus.WithDispatchers(map[string]eventbus.Dispatcher{
			"prio": eventbus.DispatchFunc(func(e eventbus.Event, hs []eventbus.Handler) {
				namedHit.Add(1)
				for _, h := range hs {
					h(e)
				}
			}),
		}),
	)

	done := make(chan struct{}, 1)
	bus.On("t", func(event eventbus.Event) { done <- struct{}{} })

	bus.Emit("t", eventbus.Event{Dispatcher: "prio"})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler not called")
	}

	if namedHit.Load() != 1 {
		t.Fatalf("expected named dispatcher called, got %d", namedHit.Load())
	}
	if selectorHit.Load() != 0 {
		t.Fatalf("selector should NOT be called when named dispatcher specified, got %d", selectorHit.Load())
	}
}

func TestNamedDispatcher_WithMiddleware(t *testing.T) {
	var mwHit atomic.Int64

	bus := eventbus.NewEventBus(
		eventbus.WithDispatchers(map[string]eventbus.Dispatcher{
			"mwtest": eventbus.SyncDispatcher{},
		}),
		eventbus.WithMiddleware(func(next eventbus.Dispatcher) eventbus.Dispatcher {
			return eventbus.DispatchFunc(func(e eventbus.Event, hs []eventbus.Handler) {
				mwHit.Add(1)
				next.Dispatch(e, hs)
			})
		}),
	)

	done := make(chan struct{}, 1)
	bus.On("t", func(event eventbus.Event) { done <- struct{}{} })

	bus.Emit("t", eventbus.Event{Dispatcher: "mwtest"})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler not called")
	}

	if mwHit.Load() != 1 {
		t.Fatalf("middleware should be called for named dispatcher, got %d", mwHit.Load())
	}
}

// ---------- 默认 SyncDispatcher fallback 测试 ----------

func TestEventBus_DefaultSyncFallback(t *testing.T) {
	bus := eventbus.NewEventBus() // 无任何配置

	done := make(chan struct{}, 1)
	bus.On("t", func(event eventbus.Event) { done <- struct{}{} })

	bus.Emit("t", eventbus.Event{})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("default SyncDispatcher should be used")
	}
}

// ---------- 通配符匹配测试 ----------

func TestWildcard_Star(t *testing.T) {
	bus := eventbus.NewEventBus()

	var count atomic.Int64
	bus.On("*", func(event eventbus.Event) {
		count.Add(1)
	})

	bus.Emit("order.created", eventbus.Event{})
	bus.Emit("user.updated", eventbus.Event{})
	bus.Emit("random", eventbus.Event{})

	if c := count.Load(); c != 3 {
		t.Fatalf("expected 3 calls, got %d", c)
	}
}

func TestWildcard_Prefix(t *testing.T) {
	bus := eventbus.NewEventBus()

	var orderCount, userCount atomic.Int64
	bus.On("order.*", func(event eventbus.Event) {
		orderCount.Add(1)
	})
	bus.On("user.*", func(event eventbus.Event) {
		userCount.Add(1)
	})

	bus.Emit("order.created", eventbus.Event{})
	bus.Emit("order.updated", eventbus.Event{})
	bus.Emit("user.login", eventbus.Event{})

	if orderCount.Load() != 2 {
		t.Fatalf("expected 2 order calls, got %d", orderCount.Load())
	}
	if userCount.Load() != 1 {
		t.Fatalf("expected 1 user call, got %d", userCount.Load())
	}
}

func TestWildcard_Suffix(t *testing.T) {
	bus := eventbus.NewEventBus()

	var deleted atomic.Int64
	bus.On("*.deleted", func(event eventbus.Event) {
		deleted.Add(1)
	})

	bus.Emit("order.deleted", eventbus.Event{})
	bus.Emit("user.deleted", eventbus.Event{})
	bus.Emit("order.created", eventbus.Event{}) // should not match

	if deleted.Load() != 2 {
		t.Fatalf("expected 2 deleted calls, got %d", deleted.Load())
	}
}

func TestWildcard_ExactAndWildcard(t *testing.T) {
	bus := eventbus.NewEventBus()

	var exact, wildcard atomic.Int64
	bus.On("order.created", func(event eventbus.Event) {
		exact.Add(1)
	})
	bus.On("order.*", func(event eventbus.Event) {
		wildcard.Add(1)
	})

	// Emit order.created → 触发 exact + wildcard
	bus.Emit("order.created", eventbus.Event{})

	if exact.Load() != 1 {
		t.Fatalf("exact handler should be called once, got %d", exact.Load())
	}
	if wildcard.Load() != 1 {
		t.Fatalf("wildcard handler should be called once, got %d", wildcard.Load())
	}

	// Emit order.updated → 只触发 wildcard
	bus.Emit("order.updated", eventbus.Event{})

	if exact.Load() != 1 {
		t.Fatalf("exact handler should still be 1, got %d", exact.Load())
	}
	if wildcard.Load() != 2 {
		t.Fatalf("wildcard handler should be 2, got %d", wildcard.Load())
	}
}

func TestWildcard_OffWildcard(t *testing.T) {
	bus := eventbus.NewEventBus()

	var count atomic.Int64
	bus.On("order.*", func(event eventbus.Event) {
		count.Add(1)
	})

	bus.Emit("order.test", eventbus.Event{})
	if count.Load() != 1 {
		t.Fatal("wildcard should match")
	}

	bus.Off("order.*")
	bus.Emit("order.test", eventbus.Event{})

	if count.Load() != 1 {
		t.Fatalf("wildcard handler should have been removed, got %d", count.Load())
	}
}

func TestWildcard_SegmentStar(t *testing.T) {
	bus := eventbus.NewEventBus()

	var count atomic.Int64
	bus.On("order.*.updated", func(event eventbus.Event) {
		count.Add(1)
	})

	bus.Emit("order.created.updated", eventbus.Event{})
	bus.Emit("order.deleted.updated", eventbus.Event{})
	bus.Emit("order.created.deleted", eventbus.Event{}) // should not match

	if count.Load() != 2 {
		t.Fatalf("expected 2 calls, got %d", count.Load())
	}
}
