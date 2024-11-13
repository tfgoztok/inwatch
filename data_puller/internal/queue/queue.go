package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"data_puller/internal/model"
)

// Work represents a unit of work to be processed.
type Work struct {
	Device *model.Device
	Query  *model.DeviceQuery
}

// ScheduledWork represents work that is scheduled to run at a specific interval.
type ScheduledWork struct {
	Work     *Work
	Interval time.Duration
	NextRun  time.Time
	StopChan chan struct{}
}

// WorkQueue manages a queue of work items and their scheduling.
type WorkQueue struct {
	mu           sync.RWMutex
	workChan     chan *Work
	scheduled    map[string]*ScheduledWork // key: deviceID_parameterName
	workerCount  int
	maxQueueSize int
}

// NewWorkQueue initializes a new WorkQueue with the specified worker count and max queue size.
func NewWorkQueue(workerCount, maxQueueSize int) *WorkQueue {
	return &WorkQueue{
		workChan:     make(chan *Work, maxQueueSize),
		scheduled:    make(map[string]*ScheduledWork),
		workerCount:  workerCount,
		maxQueueSize: maxQueueSize,
	}
}

// Start begins the scheduler and worker goroutines.
func (q *WorkQueue) Start(ctx context.Context) error {
	// Start scheduler
	go q.scheduler(ctx)

	// Start workers
	for i := 0; i < q.workerCount; i++ {
		go q.worker(ctx, i)
	}

	return nil
}

// worker processes work items from the work channel.
func (q *WorkQueue) worker(ctx context.Context, id int) {
	for {
		select {
		case <-ctx.Done():
			return
		case work := <-q.workChan:
			// İşi gerçekleştirecek handler'a gönder
			// Bu kısım Collector tarafından implement edilecek
		}
	}
}

// scheduler checks for scheduled work and enqueues it for processing.
func (q *WorkQueue) scheduler(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			q.checkScheduledWork(ctx)
		}
	}
}

// checkScheduledWork checks if any scheduled work is due to run.
func (q *WorkQueue) checkScheduledWork(ctx context.Context) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	now := time.Now()
	for _, sw := range q.scheduled {
		if now.After(sw.NextRun) {
			select {
			case <-ctx.Done():
				return
			case q.workChan <- sw.Work:
				sw.NextRun = now.Add(sw.Interval)
			default:
				// Queue is full, skip this iteration
			}
		}
	}
}

// Schedule adds a new work item to the schedule with a specified interval.
func (q *WorkQueue) Schedule(ctx context.Context, work *Work, interval time.Duration) error {
	if interval <= 0 {
		return fmt.Errorf("invalid interval: %v", interval)
	}

	key := fmt.Sprintf("%d_%s", work.Device.ID, work.Query.ParameterName)

	q.mu.Lock()
	defer q.mu.Unlock()

	// Cancel existing schedule if any
	if existing, ok := q.scheduled[key]; ok {
		close(existing.StopChan)
		delete(q.scheduled, key)
	}

	q.scheduled[key] = &ScheduledWork{
		Work:     work,
		Interval: interval,
		NextRun:  time.Now(),
		StopChan: make(chan struct{}),
	}

	return nil
}

// Cancel removes a scheduled work item based on device ID and parameter name.
func (q *WorkQueue) Cancel(deviceID int64, parameterName string) error {
	key := fmt.Sprintf("%d_%s", deviceID, parameterName)

	q.mu.Lock()
	defer q.mu.Unlock()

	if sw, ok := q.scheduled[key]; ok {
		close(sw.StopChan)
		delete(q.scheduled, key)
	}

	return nil
}

// CancelDevice cancels all scheduled work items for a specific device.
func (q *WorkQueue) CancelDevice(deviceID int64) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for key, sw := range q.scheduled {
		if sw.Work.Device.ID == deviceID {
			close(sw.StopChan)
			delete(q.scheduled, key)
		}
	}

	return nil
}

// GetScheduledWork returns a list of all scheduled work items.
func (q *WorkQueue) GetScheduledWork() []*ScheduledWork {
	q.mu.RLock()
	defer q.mu.RUnlock()

	result := make([]*ScheduledWork, 0, len(q.scheduled))
	for _, sw := range q.scheduled {
		result = append(result, sw)
	}

	return result
}

// QueueSize returns the current size of the work queue.
func (q *WorkQueue) QueueSize() int {
	return len(q.workChan)
}

// ScheduledCount returns the number of scheduled work items.
func (q *WorkQueue) ScheduledCount() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.scheduled)
}
