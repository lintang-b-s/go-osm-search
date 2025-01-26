package concurrent

import "sync"

type BackgroundWorker[T JobI, G any] struct {
	workers   int
	msgC      chan T
	waitGroup sync.WaitGroup
	jobFunc   JobFunc[T, G]
}

func NewBackgroundWorker[T JobI, G any](workers, buffer int, jobFunc JobFunc[T, G]) *BackgroundWorker[T, G] {
	return &BackgroundWorker[T, G]{
		workers: workers,
		msgC:    make(chan T, buffer),
		jobFunc: jobFunc,
	}
}

func (bw *BackgroundWorker[JobI, any]) TiggerProcessing(jobData JobI) {
	bw.msgC <- jobData
}

func (bw *BackgroundWorker[JobI, any]) Start() {

	bw.waitGroup.Add(bw.workers)
	for i := 0; i < bw.workers; i++ {
		go func() {
			for {
				select {
				case jobData, open := <-bw.msgC:
					if !open {
						bw.waitGroup.Done()
						return
					}
					// process
					bw.jobFunc(jobData)
				}
			}
		}()
	}
}

func (bw *BackgroundWorker[T, G]) Close() {
	close(bw.msgC)
	bw.waitGroup.Wait()
}
