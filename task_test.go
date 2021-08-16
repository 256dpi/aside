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

func Test(t *testing.T) {
	ch1 := make(chan error, 1)
	ch2 := make(chan error)
	defer close(ch1)
	defer close(ch2)

	task := New(func(done func(stop func() error)) error {
		select {
		case err := <-ch1:
			return err
		default:
			done(nil)
		}

		return <-ch2
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

	time.Sleep(10 * time.Millisecond)

	assert.True(t, task.Running())
	started, err = task.Verify(true)
	assert.False(t, started)
	assert.NoError(t, err)
	assert.True(t, task.Running())

	/* task return */

	ch2 <- nil
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

	/* task error */

	ch2 <- io.EOF
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

	/* boot error */

	ch2 <- nil
	ch1 <- io.EOF
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
