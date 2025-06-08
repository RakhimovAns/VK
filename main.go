package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

type Config struct {
	WorkerCount    int `json:"worker_count"`
	TaskBufferSize int `json:"task_buffer_size"`
}

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
				time.Sleep(500 * time.Millisecond)
			case <-ctx.Done():
				fmt.Printf("Worker #%d received shutdown signal\n", w.id)
				return
			}
		}
	}()
}

type WorkerPool struct {
	tasks   chan Task
	workers []*Worker
	mu      sync.Mutex
	nextID  int
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewWorkerPool(ctx context.Context, bufferSize int) *WorkerPool {
	ctx, cancel := context.WithCancel(ctx)
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

func loadConfig(path string) (Config, error) {
	var cfg Config
	file, err := os.Open(path)
	if err != nil {
		return cfg, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&cfg)
	return cfg, err
}

func main() {
	cfg, err := loadConfig("config.json")
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	ctx := context.Background()
	pool := NewWorkerPool(ctx, cfg.TaskBufferSize)

	for i := 0; i < cfg.WorkerCount; i++ {
		pool.AddWorker()
	}

	for i := 0; i < 10; i++ {
		pool.Submit(Task("Task #" + strconv.Itoa(i)))
	}

	time.Sleep(3 * time.Second)
	pool.ShutdownGracefully()
}
