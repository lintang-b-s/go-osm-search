package pkg

import "sync"

type FanInFanOut[T JobI, G any] struct {
	inputs chan T
}


func NewFanInFanOut[T JobI, G any](inputsSize int) *FanInFanOut[T, G] {
	return &FanInFanOut[T, G]{
		inputs: make(chan T, inputsSize),
	}
}

func (ff *FanInFanOut[JobI, G]) GeneratePipeline(job []JobI) {
	for _, j := range job {
		ff.inputs <- j
	}
	close(ff.inputs)
}

func (ff *FanInFanOut[JobI, G]) DoJob(job JobI, jobFunc JobFunc[JobI, G]) G {
	return jobFunc(job)
}

func (ff *FanInFanOut[JobI, G]) FanOut(jobFunc JobFunc[JobI, G]) <-chan G {
	chE := make(chan G)

	go func() {
		for job := range ff.inputs {
			chE <- ff.DoJob(job, jobFunc)
		}
		close(chE)
	}()
	return chE
}

func (ff *FanInFanOut[JobI, G]) FanIn(cs ...<-chan G) <-chan G {
	var wg sync.WaitGroup

	out := make(chan G)

	send := func(c <-chan G) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}

	wg.Add(len(cs))

	for _, c := range cs {
		go send(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
