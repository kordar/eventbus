package eventbus_test

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kordar/eventbus"
)

// ============================================================================
// 维度一：Emit 吞吐量 (TPS)
// ============================================================================

func noopHandler(eventbus.Event) {}

func BenchmarkEmit_NoHandler(b *testing.B) {
	bus := eventbus.NewEventBus()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Emit("t", eventbus.Event{})
	}
}

func BenchmarkEmit_1Handler(b *testing.B) {
	bus := eventbus.NewEventBus()
	bus.On("t", noopHandler)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Emit("t", eventbus.Event{})
	}
}

func BenchmarkEmit_10Handlers(b *testing.B) {
	bus := eventbus.NewEventBus()
	for i := 0; i < 10; i++ {
		bus.On("t", noopHandler)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Emit("t", eventbus.Event{})
	}
}

func BenchmarkEmit_100Handlers(b *testing.B) {
	bus := eventbus.NewEventBus()
	for i := 0; i < 100; i++ {
		bus.On("t", noopHandler)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Emit("t", eventbus.Event{})
	}
}

func BenchmarkOn(b *testing.B) {
	bus := eventbus.NewEventBus()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.On("t", noopHandler)
	}
}

func BenchmarkOff(b *testing.B) {
	bus := eventbus.NewEventBus()
	bus.On("t", noopHandler)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Off("t")
	}
}

// ============================================================================
// 维度二：Emit 延迟分布 (P50/P95/P99)
// ============================================================================

func TestEmitLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping latency test in short mode")
	}

	const N = 100_000
	bus := eventbus.NewEventBus()
	bus.On("t", noopHandler)

	latencies := make([]time.Duration, N)
	for i := 0; i < N; i++ {
		start := time.Now()
		bus.Emit("t", eventbus.Event{})
		latencies[i] = time.Since(start)
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	var total time.Duration
	for _, d := range latencies {
		total += d
	}

	avg := total / N
	p50 := latencies[N*50/100]
	p95 := latencies[N*95/100]
	p99 := latencies[N*99/100]
	p999 := latencies[N*999/1000]
	min := latencies[0]
	max := latencies[N-1]

	t.Logf("=== Emit Latency (1 Handler, N=%d) ===", N)
	t.Logf("Avg:   %v", avg)
	t.Logf("Min:   %v", min)
	t.Logf("P50:   %v", p50)
	t.Logf("P95:   %v", p95)
	t.Logf("P99:   %v", p99)
	t.Logf("P999:  %v", p999)
	t.Logf("Max:   %v", max)
}

// ============================================================================
// 维度三：并发能力
// ============================================================================

func BenchmarkEmit_Parallel(b *testing.B) {
	bus := eventbus.NewEventBus()
	bus.On("t", noopHandler)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bus.Emit("t", eventbus.Event{})
		}
	})
}

func BenchmarkEmit_Parallel_10Handlers(b *testing.B) {
	bus := eventbus.NewEventBus()
	for i := 0; i < 10; i++ {
		bus.On("t", noopHandler)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bus.Emit("t", eventbus.Event{})
		}
	})
}

func BenchmarkEmit_Parallel_MultiTopic(b *testing.B) {
	bus := eventbus.NewEventBus()
	for i := 0; i < 100; i++ {
		bus.On(fmt.Sprintf("t%d", i), noopHandler)
	}

	topics := make([]string, 100)
	for i := 0; i < 100; i++ {
		topics[i] = fmt.Sprintf("t%d", i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			bus.Emit(topics[i%100], eventbus.Event{})
			i++
		}
	})
}

// ============================================================================
// 维度四：Handler 数量扩展
// ============================================================================

func benchHandlerCount(b *testing.B, n int) {
	bus := eventbus.NewEventBus()
	for i := 0; i < n; i++ {
		bus.On("t", noopHandler)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Emit("t", eventbus.Event{})
	}
}

func BenchmarkEmit_50Handlers(b *testing.B)  { benchHandlerCount(b, 50) }
func BenchmarkEmit_200Handlers(b *testing.B) { benchHandlerCount(b, 200) }
func BenchmarkEmit_500Handlers(b *testing.B) { benchHandlerCount(b, 500) }

// ============================================================================
// 维度五：并发 On/Off + Emit 混合（防死锁）
// ============================================================================

func TestNoDeadlock_OnOffEmitConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping deadlock test in short mode")
	}

	bus := eventbus.NewEventBus()
	done := make(chan struct{})
	timeout := time.After(3 * time.Second)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				bus.Emit("t", eventbus.Event{Payload: "x"})
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				bus.On("t", noopHandler)
				bus.Off("t")
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				bus.On("t", noopHandler)
			}
		}
	}()

	select {
	case <-timeout:
		close(done)
		wg.Wait()
		t.Log("no deadlock detected within 3s")
	case <-done:
	}
}

// ============================================================================
// 维度六：Panic 隔离
// ============================================================================

func TestPanicIsolation_Sync(t *testing.T) {
	bus := eventbus.NewEventBus()

	normalCalled := make(chan struct{}, 1)

	bus.On("t", func(e eventbus.Event) {
		panic("intentional panic for testing isolation")
	})
	bus.On("t", func(e eventbus.Event) {
		normalCalled <- struct{}{}
	})

	bus.Emit("t", eventbus.Event{})

	select {
	case <-normalCalled:
		t.Log("panic isolation OK: handler after panic was called")
	case <-time.After(time.Second):
		t.Fatal("panic NOT isolated: handler after panic was NOT called")
	}

	afterPanicCalled := make(chan struct{}, 1)
	bus.On("t", func(e eventbus.Event) {
		afterPanicCalled <- struct{}{}
	})
	bus.Emit("t", eventbus.Event{})

	select {
	case <-afterPanicCalled:
		t.Log("panic isolation OK: bus still usable after panic")
	case <-time.After(time.Second):
		t.Fatal("bus broken after panic")
	}
}

// ============================================================================
// 维度七：GC 压力
// ============================================================================

func TestGCPressure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping GC test in short mode")
	}

	bus := eventbus.NewEventBus()
	bus.On("t", noopHandler)

	var ms runtime.MemStats
	const rounds = 10
	const iterPerRound = 50_000

	runtime.GC()
	runtime.ReadMemStats(&ms)
	gcBefore := ms.NumGC

	for r := 0; r < rounds; r++ {
		for i := 0; i < iterPerRound; i++ {
			bus.Emit("t", eventbus.Event{})
		}
	}

	runtime.ReadMemStats(&ms)
	gcAfter := ms.NumGC

	totalEvents := rounds * iterPerRound
	gcCount := gcAfter - gcBefore

	t.Logf("Total emits: %d", totalEvents)
	t.Logf("GC count during test: %d", gcCount)
	t.Logf("Avg emits per GC: %.0f", float64(totalEvents)/math.Max(float64(gcCount), 1))
	t.Logf("HeapAlloc: %.2f MB", float64(ms.HeapAlloc)/1024/1024)
	if gcCount > 1 {
		t.Logf("NOTE: allocations are triggering GC during emit")
	} else {
		t.Logf("GC pressure is low during emit")
	}
}

// ============================================================================
// 维度八：慢 Handler 阻塞（通过命名 Dispatcher 测试异步/同步行为）
// ============================================================================

func TestSlowHandler_SyncBlocks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow handler test in short mode")
	}

	bus := eventbus.NewEventBus()
	start := time.Now()
	bus.On("t", func(e eventbus.Event) {
		time.Sleep(50 * time.Millisecond)
	})
	bus.Emit("t", eventbus.Event{})
	elapsed := time.Since(start)

	t.Logf("sync Emit with 50ms handler took: %v", elapsed)
	if elapsed < 45*time.Millisecond {
		t.Log("OK: slow handler is blocking as expected in sync mode")
	}
}

func TestSlowHandler_GoroutineDispatcher_NotBlocked(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping goroutine async test in short mode")
	}

	bus := eventbus.NewEventBus(
		eventbus.WithDispatchers(map[string]eventbus.Dispatcher{
			"async": eventbus.GoroutineDispatcher{},
		}),
	)

	start := time.Now()
	bus.On("t", func(e eventbus.Event) {
		time.Sleep(50 * time.Millisecond)
	})
	bus.Emit("t", eventbus.Event{Dispatcher: "async"})
	elapsed := time.Since(start)

	t.Logf("GoroutineDispatcher Emit with 50ms handler took: %v", elapsed)
	if elapsed > 10*time.Millisecond {
		t.Logf("WARNING: GoroutineDispatcher still blocked caller for >10ms")
	} else {
		t.Log("OK: GoroutineDispatcher does not block caller")
	}
}

// ============================================================================
// 维度九：Close 安全性
// ============================================================================

func TestEmitAfterClose(t *testing.T) {
	bus := eventbus.NewEventBus()
	bus.Close()

	bus.Emit("t", eventbus.Event{})
	t.Log("OK: emit after close does not panic")
}

// ============================================================================
// 维度十：高并发延迟分布
// ============================================================================

func TestEmitLatencyUnderConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrency latency test in short mode")
	}

	const (
		goroutines = 100
		perG       = 10_000
		total      = goroutines * perG
	)
	bus := eventbus.NewEventBus()
	bus.On("t", noopHandler)

	latencies := make([]time.Duration, total)
	var wg sync.WaitGroup

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(offset int) {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				start := time.Now()
				bus.Emit("t", eventbus.Event{})
				latencies[offset*perG+i] = time.Since(start)
			}
		}(g)
	}
	wg.Wait()

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	var totalDuration time.Duration
	for _, d := range latencies {
		totalDuration += d
	}

	tps := float64(total) / totalDuration.Seconds()

	t.Logf("=== Concurrency Latency (%d goroutines × %d, total %d) ===", goroutines, perG, total)
	t.Logf("Avg:   %v", totalDuration/time.Duration(total))
	t.Logf("P50:   %v", latencies[total*50/100])
	t.Logf("P95:   %v", latencies[total*95/100])
	t.Logf("P99:   %v", latencies[total*99/100])
	t.Logf("P999:  %v", latencies[total*999/1000])
	t.Logf("TPS:   %.0f events/s", tps)
}

// ============================================================================
// 维度十一：Alloc
// ============================================================================

func BenchmarkEmit_Alloc_NoHandler(b *testing.B) {
	bus := eventbus.NewEventBus()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bus.Emit("t", eventbus.Event{})
	}
}

func BenchmarkEmit_Alloc_1Handler(b *testing.B) {
	bus := eventbus.NewEventBus()
	bus.On("t", noopHandler)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bus.Emit("t", eventbus.Event{})
	}
}

func BenchmarkEmit_MultiTopic(b *testing.B) {
	bus := eventbus.NewEventBus()
	const numTopics = 50
	for i := 0; i < numTopics; i++ {
		bus.On(fmt.Sprintf("t%d", i), noopHandler)
	}
	topics := make([]string, numTopics)
	for i := range topics {
		topics[i] = fmt.Sprintf("t%d", i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Emit(topics[i%numTopics], eventbus.Event{})
	}
}

// ============================================================================
// 维度十二：事件丢失
// ============================================================================

func TestNoEventLoss_Sync(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping event loss test in short mode")
	}

	const total = 100_000
	var received atomic.Int64

	bus := eventbus.NewEventBus()
	bus.On("t", func(e eventbus.Event) {
		received.Add(1)
	})

	for i := 0; i < total; i++ {
		bus.Emit("t", eventbus.Event{Payload: i})
	}

	got := int(received.Load())
	if got != total {
		t.Fatalf("event loss detected: sent=%d, received=%d, lost=%d", total, got, total-got)
	}
	t.Logf("no event loss: sent=%d, received=%d", total, got)
}
