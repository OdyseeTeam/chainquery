package util

import "sync"

// PieceOfWork is an interface type for representing a atomic piece of work that can be done independently.
type PieceOfWork interface {
	// BeforeExecute will always be executed before the piece of work's execution.
	BeforeExecute()
	// Execute is the execution of the actual work.
	Execute() error
	// AfterExecute will always be executed after the piece of work's execution if successful.
	AfterExecute()
	// OnError will execute in the event an error is returned from the Execute function.
	OnError(err error)
}

// InitWorkers creates a worker pool that execute pieces of work. It is a way of controlling the number go routines to
// optimize parallelism. It is recommended that this stay around the number of cores unless there is significant blocking
// time associated with the work involved.
func InitWorkers(numworkers int, jobs chan PieceOfWork) *sync.WaitGroup {
	wg := sync.WaitGroup{}
	for i := 0; i < numworkers; i++ {
		wg.Add(1)
		go func() {
			for job := range jobs {
				err := job.Execute()
				job.OnError(err)
			}
			wg.Done()
		}()
	}

	return &wg
}

// NewQueue creates a bi-directional channel that can take in pieces of work. This is leveraged with the worker pool
// and is what they pull from while active. The worker pool will end once this channel is closed. The intention is that
// this channel will be passed into the initialization of the worker pool.
func NewQueue() chan PieceOfWork {
	return make(chan PieceOfWork)
}
