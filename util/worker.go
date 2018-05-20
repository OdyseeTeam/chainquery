package util

import "sync"

type PieceOfWork interface {
	BeforeExecute()
	Execute() error
	AfterExecute()
	OnError(err error)
}

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

func NewQueue() chan PieceOfWork {
	return make(chan PieceOfWork)
}
