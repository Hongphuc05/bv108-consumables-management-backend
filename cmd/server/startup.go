package main

import (
	"log"
	"sync"
	"time"
)

type startupStep struct {
	name string
	run  func() error
}

func mustRunStartupStep(stepName string, fn func() error) {
	startedAt := time.Now()
	if err := fn(); err != nil {
		log.Fatalf("%s failed: %v", stepName, err)
	}
	log.Printf("[startup] %s completed in %s", stepName, time.Since(startedAt).Round(time.Millisecond))
}

func mustRunStartupStepsParallel(steps ...startupStep) {
	if len(steps) == 0 {
		return
	}

	type startupError struct {
		name string
		err  error
	}

	var wg sync.WaitGroup
	errCh := make(chan startupError, len(steps))
	for _, step := range steps {
		step := step
		wg.Add(1)

		go func() {
			defer wg.Done()

			startedAt := time.Now()
			if err := step.run(); err != nil {
				errCh <- startupError{name: step.name, err: err}
				return
			}
			log.Printf("[startup] %s completed in %s", step.name, time.Since(startedAt).Round(time.Millisecond))
		}()
	}

	wg.Wait()
	close(errCh)
	if startupErr, ok := <-errCh; ok {
		log.Fatalf("%s failed: %v", startupErr.name, startupErr.err)
	}
}
