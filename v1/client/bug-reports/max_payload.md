### Max Payload encounter with NetCat

If you'd run the server as 
```bash
go run main.go
```
And then run `netcat`
```bash
nc localhost 4000
```
When you send in a stream of text, or anything
at all, that's enough to damage the whole server.

Decided to give the code a little nudge by running
it on a [Digital Ocean](https://digitalocean) server
and I kept getting: ```fatal error: runtime: out of memory```
I was able to preserver the stack trace of the errors
as is in ```stacktrace_max_payload_```. 

For entry of 'rand' on netcat is enough to be parsed 
as ```1919378276```, thats 1.9GiB

Hence, we're limiting the max payload to be 1MB -> 1024 * 1024 
