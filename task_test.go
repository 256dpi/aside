package aside

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTask(t *testing.T) {
	ch := make(chan error)
	defer close(ch)
	task := New(func(done func(stop func() error)) error {
		done(nil)
		return <-ch
	})

	assert.False(t, task.Running())
	started, err := task.Verify(false)
	assert.False(t, started)
	assert.NoError(t, err)
	assert.False(t, task.Running())

	assert.False(t, task.Running())
	started, err = task.Verify(true)
	assert.True(t, started)
	assert.NoError(t, err)
	assert.True(t, task.Running())

	assert.True(t, task.Running())
	started, err = task.Verify(true)
	assert.False(t, started)
	assert.NoError(t, err)
	assert.True(t, task.Running())

	assert.PanicsWithValue(t, "aside: missing stop function", func() {
		_ = task.Stop()
	})
}

func TestTaskReturn(t *testing.T) {
	ch := make(chan error)
	defer close(ch)
	task := New(func(done func(stop func() error)) error {
		done(nil)
		return <-ch
	})

	assert.False(t, task.Running())
	started, err := task.Verify(true)
	assert.True(t, started)
	assert.NoError(t, err)
	assert.True(t, task.Running())

	ch <- nil
	time.Sleep(10 * time.Millisecond)

	assert.False(t, task.Running())
	started, err = task.Verify(true)
	assert.True(t, started)
	assert.NoError(t, err)
	assert.True(t, task.Running())
}

func TestTaskError(t *testing.T) {
	ch := make(chan error)
	defer close(ch)
	task := New(func(done func(stop func() error)) error {
		done(nil)
		return <-ch
	})

	assert.False(t, task.Running())
	started, err := task.Verify(true)
	assert.True(t, started)
	assert.NoError(t, err)
	assert.True(t, task.Running())

	ch <- io.EOF
	time.Sleep(10 * time.Millisecond)

	assert.False(t, task.Running())
	started, err = task.Verify(true)
	assert.False(t, started)
	assert.Error(t, err)
	assert.False(t, task.Running())

	time.Sleep(10 * time.Millisecond)

	assert.False(t, task.Running())
	started, err = task.Verify(true)
	assert.True(t, started)
	assert.NoError(t, err)
	assert.True(t, task.Running())

	time.Sleep(10 * time.Millisecond)

	assert.True(t, task.Running())
	started, err = task.Verify(true)
	assert.False(t, started)
	assert.NoError(t, err)
	assert.True(t, task.Running())
}

func TestTaskStartError(t *testing.T) {
	start := make(chan error, 1)
	defer close(start)

	ch := make(chan error)
	defer close(ch)
	task := New(func(done func(stop func() error)) error {
		select {
		case err := <-start:
			return err
		default:
			done(nil)
		}

		return <-ch
	})

	assert.False(t, task.Running())
	started, err := task.Verify(true)
	assert.True(t, started)
	assert.NoError(t, err)
	assert.True(t, task.Running())

	ch <- nil
	start <- io.EOF
	time.Sleep(10 * time.Millisecond)

	assert.False(t, task.Running())
	started, err = task.Verify(true)
	assert.True(t, started)
	assert.Error(t, err)
	assert.False(t, task.Running())

	time.Sleep(10 * time.Millisecond)

	assert.False(t, task.Running())
	started, err = task.Verify(true)
	assert.True(t, started)
	assert.NoError(t, err)
	assert.True(t, task.Running())

	time.Sleep(10 * time.Millisecond)

	assert.True(t, task.Running())
	started, err = task.Verify(true)
	assert.False(t, started)
	assert.NoError(t, err)
	assert.True(t, task.Running())
}

func TestStopabbleTask(t *testing.T) {
	task := New(func(done func(stop func() error)) error {
		ch := make(chan error)
		done(func() error {
			close(ch)
			return nil
		})
		return <-ch
	})

	assert.False(t, task.Running())
	started, err := task.Verify(false)
	assert.False(t, started)
	assert.NoError(t, err)
	assert.False(t, task.Running())

	assert.False(t, task.Running())
	started, err = task.Verify(true)
	assert.True(t, started)
	assert.NoError(t, err)
	assert.True(t, task.Running())

	assert.True(t, task.Running())
	started, err = task.Verify(true)
	assert.False(t, started)
	assert.NoError(t, err)
	assert.True(t, task.Running())

	assert.True(t, task.Running())
	err = task.Stop()
	assert.NoError(t, err)
	assert.False(t, task.Running())

	assert.NoError(t, task.Stop())
}

func TestStopabbleTaskStopError(t *testing.T) {
	stop := make(chan error, 1)
	defer close(stop)

	ch := make(chan error)
	task := New(func(done func(stop func() error)) error {
		done(func() error {
			select {
			case err := <-stop:
				return err
			default:
				close(ch)
				return nil
			}
		})
		return <-ch
	})

	assert.False(t, task.Running())
	started, err := task.Verify(true)
	assert.True(t, started)
	assert.NoError(t, err)
	assert.True(t, task.Running())

	stop <- io.EOF
	assert.True(t, task.Running())
	err = task.Stop()
	assert.Error(t, err)
	assert.True(t, task.Running())

	assert.True(t, task.Running())
	err = task.Stop()
	assert.NoError(t, err)
	assert.False(t, task.Running())
}

func TestStopabbleTaskExitError(t *testing.T) {
	stop := make(chan error, 1)
	defer close(stop)

	ch := make(chan error)
	task := New(func(done func(stop func() error)) error {
		done(func() error {
			ch <- <-stop
			return nil
		})
		return <-ch
	})

	assert.False(t, task.Running())
	started, err := task.Verify(true)
	assert.True(t, started)
	assert.NoError(t, err)
	assert.True(t, task.Running())

	stop <- io.EOF
	assert.True(t, task.Running())
	err = task.Stop()
	assert.Error(t, err)
	assert.False(t, task.Running())

	assert.False(t, task.Running())
	err = task.Stop()
	assert.NoError(t, err)
	assert.False(t, task.Running())
}

func Example() {
	// create server task
	server := New(func(cb func(func() error)) error {
		// create socket
		socket, err := net.Listen("tcp", "0.0.0.0:1337")
		if err != nil {
			return err
		}

		// signal start
		cb(socket.Close)

		// run server
		err = http.Serve(socket, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("Hello world!"))
		}))
		if err != nil && !errors.Is(err, net.ErrClosed) {
			return err
		}

		return nil
	})

	// verify server
	started, err := server.Verify(true)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Started: %t\n", started)

	// query server
	res, err := http.Get("http://0.0.0.0:1337")
	if err != nil {
		panic(err)
	}
	buf, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(buf))

	// check state
	ok := server.Running()
	fmt.Printf("Running: %t\n", ok)

	// stop
	err = server.Stop()
	if err != nil {
		panic(err)
	}

	// check state
	ok = server.Running()
	fmt.Printf("Running: %t\n", ok)

	// verify server
	started, err = server.Verify(false)
	fmt.Printf("Started: %t\n", started)

	// Output:
	// Started: true
	// Hello world!
	// Running: true
	// Running: false
	// Started: false
}
