package metrics_queue

import (
	"log/slog"
	"sync"
	"time"

	"github.com/getmiroir/trading-service/internal/clients"
	"github.com/getmiroir/trading-service/internal/logging"
	"github.com/getmiroir/trading-service/internal/metrics/metric_types"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

// MetricsQueue defines the interface for a metrics queue.
// Implementations can be in-memory (channel-based) or persistent (file/database).
type MetricsQueue interface {
	// Push adds metrics to the queue. Non-blocking for in-memory implementations.
	Push(metrics *metric_types.LoopMetrics)
	// Start begins the background worker that pushes metrics to the API.
	Start()
	// Stop gracefully shuts down the queue and flushes pending metrics.
	Stop()
}

// InMemoryMetricsQueue implements MetricsQueue using a bounded slice.
// When full, it drops the oldest element to make room for new ones.
type InMemoryMetricsQueue struct {
	queue      []*metric_types.LoopMetrics
	maxSize    int
	controlApi *clients.ControlApi
	logger     *slog.Logger
	done       chan struct{}
	wg         sync.WaitGroup
	mu         sync.Mutex

	// Configuration
	batchSize     int
	flushInterval time.Duration
	maxRetryDelay time.Duration
	currentDelay  time.Duration
}

// NewInMemoryMetricsQueue creates a new in-memory metrics queue.
func NewInMemoryMetricsQueue(controlApi *clients.ControlApi, cfg subsettings.Metrics) *InMemoryMetricsQueue {
	bufferSize := cfg.BufferSize
	if bufferSize <= 0 {
		bufferSize = 100
	}
	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 10
	}
	flushInterval := time.Duration(cfg.FlushIntervalSeconds * float64(time.Second))
	if flushInterval <= 0 {
		flushInterval = 5 * time.Second
	}
	maxRetryDelay := time.Duration(cfg.MaxRetryDelaySeconds * float64(time.Second))
	if maxRetryDelay <= 0 {
		maxRetryDelay = 60 * time.Second
	}

	return &InMemoryMetricsQueue{
		queue:         make([]*metric_types.LoopMetrics, 0, bufferSize),
		maxSize:       bufferSize,
		controlApi:    controlApi,
		logger:        logging.NewLogger("metrics-queue"),
		done:          make(chan struct{}),
		batchSize:     batchSize,
		flushInterval: flushInterval,
		maxRetryDelay: maxRetryDelay,
		currentDelay:  flushInterval,
	}
}

// Push adds metrics to the queue. If full, drops the oldest element.
func (q *InMemoryMetricsQueue) Push(metrics *metric_types.LoopMetrics) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queue) >= q.maxSize && q.maxSize > 0 {
		to_drop := len(q.queue) - q.maxSize + 1
		dropped := q.queue[0:to_drop]
		q.queue = q.queue[to_drop:]
		q.logger.Info("Queue full, dropping oldest metrics", "dropped_id", dropped)
	}

	q.queue = append(q.queue, metrics)
	q.logger.Info("Queued metrics", "id", metrics.ID)
}

// Start begins the background worker.
func (q *InMemoryMetricsQueue) Start() {
	q.wg.Add(1)
	go q.worker()
}

// Stop gracefully shuts down the queue.
func (q *InMemoryMetricsQueue) Stop() {
	close(q.done)
	q.wg.Wait()
}

func (q *InMemoryMetricsQueue) worker() {
	defer q.wg.Done()

	timer := time.NewTimer(q.currentDelay)
	defer timer.Stop()

	for {
		select {
		case <-q.done:
			q.flush()
			return
		case <-timer.C:
			q.flush()
			timer.Reset(q.currentDelay)
		}
	}
}

func (q *InMemoryMetricsQueue) flush() {
	for {
		q.mu.Lock()
		if len(q.queue) == 0 {
			q.mu.Unlock()
			return
		}

		// Take up to batchSize items
		n := q.batchSize
		if len(q.queue) < n {
			n = len(q.queue)
		}
		batch := make([]*metric_types.LoopMetrics, n)
		copy(batch, q.queue[:n])
		q.queue = q.queue[n:]
		q.mu.Unlock()

		if err := q.push(batch); err != nil {
			q.logger.Error("Failed to push metrics", "count", len(batch), "error", err)

			// Re-queue failed metrics at the front
			q.mu.Lock()
			q.queue = append(batch, q.queue...)
			// Trim if over max size (drop oldest)
			if len(q.queue) > q.maxSize {
				q.queue = q.queue[len(q.queue)-q.maxSize:]
			}
			q.mu.Unlock()

			// Exponential backoff
			q.currentDelay *= 2
			if q.currentDelay > q.maxRetryDelay {
				q.currentDelay = q.maxRetryDelay
			}
			q.logger.Info("Next retry scheduled", "delay", q.currentDelay)
			return
		}

		// Reset delay on success
		q.currentDelay = q.flushInterval
	}
}

func (q *InMemoryMetricsQueue) push(batch []*metric_types.LoopMetrics) error {
	metrics := make([]metric_types.LoopMetrics, len(batch))
	for i, m := range batch {
		metrics[i] = *m
	}

	received, err := q.controlApi.PushLoopMetrics(metrics)
	if err != nil {
		return err
	}

	q.logger.Info("Pushed metrics successfully", "count", received)
	return nil
}
