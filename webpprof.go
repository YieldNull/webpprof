package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/google/pprof/driver"
	"github.com/gorilla/mux"
	_ "net/http/pprof"
)

type PProfServer struct {
	router *mux.Router
	mu     sync.RWMutex
	pid    int64

	profIDs      map[int64]string                 // profile id -> profile name
	profHandlers map[int64]*driver.HTTPServerArgs // profile id -> handler
}

func (s *PProfServer) profName(hostAndPort, profile string) string {
	return fmt.Sprintf("%s/%s", hostAndPort, profile)
}

func (s *PProfServer) createPProf(hostAndPort, profile string) (pid int64, err error) {
	pid = atomic.AddInt64(&s.pid, 1)

	address := fmt.Sprintf("http://%s/debug/pprof/%s", hostAndPort, profile)
	args := []string{"-http", ":8888", "--no_browser", address}

	err = driver.PProf(&driver.Options{
		Flagset: NewGoFlags(args),
		HTTPServer: func(args *driver.HTTPServerArgs) error {
			s.mu.Lock()
			s.profIDs[pid] = s.profName(hostAndPort, profile)
			s.profHandlers[pid] = args
			s.mu.Unlock()
			return nil
		},
	})
	return
}

func (s *PProfServer) deletePProf(pid int64) {
	s.mu.Lock()
	s.profHandlers[pid].Handlers = nil
	delete(s.profIDs, pid)
	delete(s.profHandlers, pid)
	s.mu.Unlock()
}

func writeErr(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err.Error()))
}

func writeText(w http.ResponseWriter, text string) {
	w.Write([]byte(text))
}

func (s *PProfServer) HandleCreate(w http.ResponseWriter, r *http.Request) {
	hostAndPort := mux.Vars(r)["hostAndPort"]
	profile := mux.Vars(r)["profile"]

	pid, err := s.createPProf(hostAndPort, profile)
	if err != nil {
		writeErr(w, err)
		return
	}

	writeText(w, strconv.FormatInt(pid, 10))
}

func (s *PProfServer) HandleDelete(w http.ResponseWriter, r *http.Request) {
	hostAndPort := mux.Vars(r)["hostAndPort"]
	profile := mux.Vars(r)["profile"]
	pidStr := mux.Vars(r)["pid"]

	pid, err := strconv.ParseInt(pidStr, 10, 64)
	if err != nil {
		writeErr(w, fmt.Errorf("invalid pid %s, an integer expected", pidStr))
		return
	}
	var expectedProfileName string
	s.mu.RLock()
	expectedProfileName = s.profIDs[pid]
	s.mu.RUnlock()
	if expectedProfileName != s.profName(hostAndPort, profile) {
		writeErr(w, fmt.Errorf("pid %s is not bound to %s/%s", pidStr, hostAndPort, profile))
		return
	}

	s.deletePProf(pid)
	writeText(w, "OK")
}

func (s *PProfServer) HandlePProfUI(w http.ResponseWriter, r *http.Request) {
	hostAndPort := mux.Vars(r)["hostAndPort"]
	profile := mux.Vars(r)["profile"]
	pidStr := mux.Vars(r)["pid"]
	rest := mux.Vars(r)["rest"]

	pid, err := strconv.ParseInt(pidStr, 10, 64)
	if err != nil {
		writeErr(w, fmt.Errorf("invalid pid %s, an integer expected", pidStr))
		return
	}

	var args *driver.HTTPServerArgs
	var expectedProfileName string
	s.mu.RLock()
	expectedProfileName = s.profIDs[pid]
	args = s.profHandlers[pid]
	s.mu.RUnlock()

	if args == nil {
		writeErr(w, errors.New("pid is not running, you should start a new prof instead"))
		return
	}
	if expectedProfileName != s.profName(hostAndPort, profile) {
		writeErr(w, fmt.Errorf("pid %s is not bound to %s/%s", pidStr, hostAndPort, profile))
		return
	}

	h := args.Handlers["/"+rest]
	if h == nil {
		h = http.DefaultServeMux
	}
	h.ServeHTTP(w, r)
}

func NewPProfServer() *PProfServer {
	s := &PProfServer{
		router:       mux.NewRouter(),
		profIDs:      make(map[int64]string),
		profHandlers: make(map[int64]*driver.HTTPServerArgs),
	}

	return s
}

func main() {
	pprofServer := NewPProfServer()
	pprofRouter := mux.NewRouter()
	pprofRouter.Path("/{hostAndPort}/{profile}/create").HandlerFunc(pprofServer.HandleCreate)
	pprofRouter.Path("/{hostAndPort}/{profile}/delete/{pid:[0-9]+}").HandlerFunc(pprofServer.HandleDelete)
	pprofRouter.Path("/{hostAndPort}/{profile}/{pid:[0-9]+}/").HandlerFunc(pprofServer.HandlePProfUI)
	pprofRouter.Path("/{hostAndPort}/{profile}/{pid:[0-9]+}/{rest}").HandlerFunc(pprofServer.HandlePProfUI)

	router := mux.NewRouter()
	router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
	router.PathPrefix("/pprof/").Handler(http.StripPrefix("/pprof", pprofRouter))

	addr := os.Args[1]
	fmt.Printf("listening on %s \n", addr)
	http.ListenAndServe(addr, router)
}

type GoFlags struct {
	*flag.FlagSet

	usageMsgs []string
	arguments []string
}

func NewGoFlags(args []string) *GoFlags {
	return &GoFlags{
		FlagSet:   flag.NewFlagSet("pprof", flag.ExitOnError),
		arguments: args,
	}
}

func (f *GoFlags) StringList(o, d, c string) *[]*string {
	return &[]*string{f.FlagSet.String(o, d, c)}
}

func (f *GoFlags) ExtraUsage() string {
	return strings.Join(f.usageMsgs, "\n")
}

func (f *GoFlags) AddExtraUsage(eu string) {
	f.usageMsgs = append(f.usageMsgs, eu)
}

func (f *GoFlags) Parse(usage func()) []string {
	f.FlagSet.Usage = usage
	_ = f.FlagSet.Parse(f.arguments)
	args := f.FlagSet.Args()
	if len(args) == 0 {
		usage()
	}
	return args
}
