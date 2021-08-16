# aside

[![Test](https://github.com/256dpi/aside/actions/workflows/test.yml/badge.svg)](https://github.com/256dpi/aside/actions/workflows/test.yml)

**Package aside implements a simple mechanism run a task on the side and join its error reporting with the primary task.**

## Example

```go
// create server task
server := New(func(cb func()) error {
    // create socket
    socket, err := net.Listen("tcp", "0.0.0.0:1337")
    if err != nil {
        return err
    }

    // signal start
    cb()

    // run closer
    done := make(chan struct{})
    go func() {
        <-done
        _ = socket.Close()
    }()

    // run server
    err = http.Serve(socket, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        _, _ = w.Write([]byte("Hello world!"))
        close(done)
    }))
    if err != nil {
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

// verify server
started, err = server.Verify(false)
fmt.Printf("Started: %t\n", started)
fmt.Printf("Error: %s\n", err)

// Output:
// Started: true
// Hello world!
// Running: false
// Started: false
// Error: accept tcp [::]:1337: use of closed network connection
```
