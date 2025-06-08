package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Task string

type Worker struct {
	id int
}

func (w *Worker) Start(ctx context.Context, tasks <-chan Task, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case task := <-tasks:
				fmt.Printf("Worker #%d processing task: %s\n", w.id, task)
				time.Sleep(500 * time.Millisecond) // имитация работы
			case <-ctx.Done():
				fmt.Printf("Worker #%d received shutdown signal\n", w.id)
				return
			}
		}
	}()
}

type WorkerPool struct {
	tasks     chan Task
	workers   []*Worker
	mu        sync.Mutex
	nextID    int
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	startOnce sync.Once
}

func NewWorkerPool(bufferSize int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		tasks:  make(chan Task, bufferSize),
		ctx:    ctx,
		cancel: cancel,
	}
}

func (p *WorkerPool) AddWorker() {
	p.mu.Lock()
	defer p.mu.Unlock()

	worker := &Worker{id: p.nextID}
	p.workers = append(p.workers, worker)
	worker.Start(p.ctx, p.tasks, &p.wg)
	fmt.Printf("Added worker #%d\n", worker.id)
	p.nextID++
}

func (p *WorkerPool) Submit(task Task) {
	select {
	case p.tasks <- task:
	case <-p.ctx.Done():
		fmt.Println("Cannot submit task, pool is shutting down")
	}
}

func (p *WorkerPool) ShutdownGracefully() {
	fmt.Println("Shutting down worker pool...")
	p.cancel()
	p.wg.Wait()
	close(p.tasks)
	fmt.Println("All workers stopped.")
}

func main() {
	pool := NewWorkerPool(10)

	pool.AddWorker()
	pool.AddWorker()

	for i := 0; i < 5; i++ {
		pool.Submit(Task(fmt.Sprintf("Task #%d", i)))
	}

	time.Sleep(2 * time.Second)

	pool.AddWorker()

	for i := 5; i < 10; i++ {
		pool.Submit(Task(fmt.Sprintf("Task #%d", i)))
	}

	time.Sleep(2 * time.Second)
	pool.ShutdownGracefully()
}
