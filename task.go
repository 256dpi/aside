// Package aside implements a simple mechanism run a task on the side and
// join its error reporting with the primary task.
package aside

import (
	"fmt"
	"sync"
)

// Task is a side task.
type Task struct {
	fn      func(func(func() error)) error
	stop    func() error
	running chan struct{}
	error   error
	mutex   sync.Mutex
}

// New will create and return new side task. The specified function is called
// to run the task. The callback should be called once the task has been
// successfully started and is running.
func New(fn func(cb func(stop func() error)) error) *Task {
	return &Task{
		fn: fn,
	}
}

// Running returns whether the task is currently running.
func (t *Task) Running() bool {
	// acquire mutex
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return t.running != nil
}

// Verify will verify that the task is running. It will start the task if
// requested and not running already. If the task stopped prior to the call the
// returned error is returned without starting the task. If the task has been
// started the second argument will be true. When starting the task the function
// will wait until the task callback has been called.
func (t *Task) Verify(start bool) (bool, error) {
	// acquire mutex
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// check error
	if t.error != nil {
		err := t.error
		t.error = nil
		return false, err
	}

	// check running
	if t.running != nil {
		return false, nil
	}

	// check start
	if !start {
		return false, nil
	}

	// set flag
	t.running = make(chan struct{})

	// prepare started
	started := make(chan struct{})

	// run task
	go func() {
		var err error
		defer func() {
			t.mutex.Lock()
			t.error = err
			close(t.running)
			t.running = nil
			if started != nil {
				close(started)
			}
			t.mutex.Unlock()
		}()
		err = t.fn(func(stop func() error) {
			t.mutex.Lock()
			t.stop = stop
			close(started)
			t.mutex.Unlock()
		})
	}()

	// await started
	t.mutex.Unlock()
	<-started
	started = nil
	t.mutex.Lock()

	// check error
	if t.error != nil {
		err := t.error
		t.error = nil
		return true, err
	}

	return true, nil
}

// Stop will stop the task if running using the supplied stop method to the
// callback.
func (t *Task) Stop() error {
	// acquire mutex
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// check state
	if t.running == nil {
		return nil
	}

	// check stop
	if t.stop == nil {
		return fmt.Errorf("missing stop")
	}

	// capture running
	running := t.running

	// stop
	err := t.stop()
	if err != nil {
		return err
	}

	// await stop
	t.mutex.Unlock()
	<-running
	t.mutex.Lock()

	// check error
	if t.error != nil {
		err := t.error
		t.error = nil
		return err
	}

	return nil
}
