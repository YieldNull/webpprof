# Introduction

webpprof runs multiple `go tool pprof` web interfaces in a web server without installing [pprof](https://github.com/google/pprof) locally.

# How to use

webpprof exposes several APIs:

- `/pprof/{hostAndPort}/{profile}/create` : create a profile
- `/{hostAndPort}/{profile}/{pid:[0-9]+}/`: access the pprof web interface
- `/{hostAndPort}/{profile}/delete/{pid:[0-9]+}`: delete a profile

For example:

1. `GET /pprof/127.0.0.1:8888/goroutine/create` which create a new profile and returns profile id `1`
2. `GET /pprof/127.0.0.1:8888/goroutine/1/` visit pprof web ui
3. `GET /pprof/127.0.0.1:8888/goroutine/delete` delete profile with id `1`

![demo.gif](demo.gif)
