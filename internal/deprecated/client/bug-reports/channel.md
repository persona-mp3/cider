### Stdin Channel
Running on macos, If you ran the client and the server is 
also running
```bash
go run internal/client/main.go
```

You can hit ```Ctrl+C``` and that kills the app. But if you hit ```Ctrl+D```,
that by definition it's supposed to trigger the same behaviour. It simply 
means ```End of Tranmission``` which, the Scanner treats as EOF but not 
as an Error which would have been nice


From the go-routine reading from the stdin (before all the debug statements you'd see)
```go
	stdin := make(chan string)
	in := os.Stdin
	scanner := bufio.NewScanner(os.Stdin)
	go func() {
		for scanner.Scan() {
			value := scanner.Text()
			select {
			case <-ctx.Done():
				fmt.Println(" [debug] ctx called in scanner", ctx.Err().Error())
				return

			case stdin <- scanner.Text()
				fmt.Print(" [*] ")
			}
		}
	}()

	slog.Info("connected to stdin successfully")
	return stdin
```

This is perfect, clear right?
And now the reader
```go
    stdin := fromStdin(ctx)
    select {
        case v := <-stdin
    }
```

And what happens when the `fromStdin` returns? A LEAK 
And now that go-routine is orphaned. No matter what you do
after hitting ```Ctrl-D``` that is going to stop the whole program
You'd have to manually grep and kill or suspend it in your current 
terminal and kill it.

And the fix?
```go
    stdin := make(chan os.Stdin)
    go func() {
        defer close(stdin)
        ....
    }()
```
Not yet

```go
    ch := fromStdin(ctx)
    select {
        v, open := <-stdin // boom
    }
```

ALWAYS CLOSE YOUR CHANNELS IF YOU'RE A CREATOR
