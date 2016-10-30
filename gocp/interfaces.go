package main

type IBackgroundProcess interface {
	Start()
	Stop()
}

func Start(procs ...IBackgroundProcess) {
	for _, proc := range procs {
		go proc.Start()
	}
}

func Stop(procs ...IBackgroundProcess) {
	for _, proc := range procs {
		proc.Stop()
	}
}
