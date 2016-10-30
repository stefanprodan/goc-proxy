package main

type Worker interface {
	Start()
	Stop()
}

func startWorkers(workers ...Worker) {
	for _, w := range workers {
		go w.Start()
	}
}

func stopWorkers(workers ...Worker) {
	for _, w := range workers {
		w.Stop()
	}
}
