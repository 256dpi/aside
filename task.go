// Package aside implements a simple mechanism run a task on the side and
// join its error reporting with the primary task.
package aside

import "sync"

// Task is a side task.
type Task struct {
	fn      func(func()) error
	running bool
	error   error
	mutex   sync.Mutex
}

// New will create and return new side task. The specified function is called
// to run the task. The callback should be called once the task has been
// successfully started and is running.
func New(fn func(cb func()) error) *Task {
	return &Task{
		fn: fn,
	}
}

// Running returns whether the task is currently running.
func (t *Task) Running() bool {
	// acquire mutex
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return t.running
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
	if t.running {
		return false, nil
	}

	// check start
	if !start {
		return false, nil
	}

	// set flag
	t.running = true

	// prepare done
	done := make(chan struct{})

	// run task
	go func() {
		var err error
		defer func() {
			t.mutex.Lock()
			t.error = err
			t.running = false
			if done != nil {
				close(done)
			}
			t.mutex.Unlock()
		}()
		err = t.fn(func() {
			t.mutex.Lock()
			close(done)
			t.mutex.Unlock()
		})
	}()

	// await done
	t.mutex.Unlock()
	<-done
	done = nil
	t.mutex.Lock()

	// check error
	if t.error != nil {
		err := t.error
		t.error = nil
		return true, err
	}

	return true, nil
}
