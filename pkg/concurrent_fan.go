package pkg

type FanInFanOut[T JobI, G any] struct {
	inputs chan T
}

type ConsumeFunc[G any] func(resChan <-chan G) error

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

func (ff *FanInFanOut[JobI, G]) DoJob(jobFunc JobFunc[JobI, G]) <-chan G {
	out := make(chan G)
	go func() {
		for job := range ff.inputs {
			out <- jobFunc(job)
		}
		close(out)
	}()
	return out
}

func (ff *FanInFanOut[JobI, G]) FanOut(goroutinesNum int, jobFunc JobFunc[JobI, G]) []<-chan G {
	outs := make([]<-chan G, goroutinesNum)
	for i := 0; i < goroutinesNum; i++ {
		outs[i] = ff.DoJob(jobFunc)
	}
	return outs
}

func (ff *FanInFanOut[JobI, G]) FanIn(consumeFunc ConsumeFunc[G], cs ...<-chan G) error {
	for _, resVal := range cs {
		err := consumeFunc(resVal)
		if err != nil {
			return err
		}
	}
	return nil
}
