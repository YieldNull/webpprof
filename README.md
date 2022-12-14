# Introduction

webpprof runs multiple `go tool pprof` web interfaces in a web server without installing [pprof](https://github.com/google/pprof) locally.

# How to use

## Start server

To build and install it:

```shell
$ go install github.com/yieldnull/webpprof@latest
```

The binary will be installed `$GOPATH/bin`.

To run it, just pass a listening address:

```shell
$ webpprof :8888
listening on :8888
```

## Create pprof

webpprof exposes several APIs:

- `/pprof/{hostAndPort}/{profile}/create` : create a profile and return the profile id
- `/pprof/{hostAndPort}/{profile}/{pid:[0-9]+}/`: access the pprof web interface by profile id
- `/pprof/{hostAndPort}/{profile}/delete/{pid:[0-9]+}`: delete a profile by profile id

`hostAndPort` is the server that you want to run pprof with, and `profile` is `goroutine`,`heap`,`profile` etc. which is listed on `/debug/pprof/`.

For example:

1. `GET /pprof/127.0.0.1:8888/goroutine/create` which create a new profile and returns profile id `1`
2. `GET /pprof/127.0.0.1:8888/goroutine/1/` visit pprof web ui
3. `GET /pprof/127.0.0.1:8888/goroutine/delete/1` delete profile with id `1`

# Demo

![demo.gif](demo.gif)
