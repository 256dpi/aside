# aside

[![Test](https://github.com/256dpi/aside/actions/workflows/test.yml/badge.svg)](https://github.com/256dpi/aside/actions/workflows/test.yml)

**A simple mechanism run a task on the side and join its error reporting with the primary task.**

## Example

```go
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
```
