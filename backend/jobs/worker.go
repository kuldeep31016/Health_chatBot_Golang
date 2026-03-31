package jobs

import "log"

type Job struct {
	ID        string
	Operation func() error
	OnSuccess func()
	OnFail    func(err error)
}

type WorkerPool struct {
	jobs    chan Job
	workers int
}

func NewWorkerPool(workers int) *WorkerPool {
	if workers <= 0 {
		workers = 1
	}

	return &WorkerPool{
		jobs:    make(chan Job, workers*4),
		workers: workers,
	}
}

func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workers; i++ {
		go func(workerID int) {
			for job := range wp.jobs {
				if job.Operation == nil {
					continue
				}

				err := job.Operation()
				if err != nil {
					if job.OnFail != nil {
						job.OnFail(err)
					} else {
						log.Printf("worker %d: job %s failed: %v", workerID, job.ID, err)
					}
					continue
				}

				if job.OnSuccess != nil {
					job.OnSuccess()
				}
			}
		}(i + 1)
	}
}

func (wp *WorkerPool) Submit(job Job) {
	wp.jobs <- job
}
