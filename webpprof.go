package main

import (
	"flag"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/pprof/driver"
	"github.com/gorilla/mux"
	_ "net/http/pprof"
)

type PProfServer struct {
	router *mux.Router
	profs  map[string]chan struct{}
}

func NewPProfServer() *PProfServer {
	s := &PProfServer{
		router: mux.NewRouter(),
		profs:  make(map[string]chan struct{}),
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		hostAndPort := mux.Vars(r)["hostAndPort"]
		profile := mux.Vars(r)["profile"]
		rest := mux.Vars(r)["rest"]
		path := "/" + rest

		args := []string{"-http", ":8888", "--no_browser", fmt.Sprintf("http://%s/debug/pprof/%s", hostAndPort, profile)}
		driver.PProf(&driver.Options{
			Flagset: NewGoFlags(args),
			HTTPServer: func(args *driver.HTTPServerArgs) error {
				h := args.Handlers[path]
				if h == nil {
					// Fall back to default behavior
					h = http.DefaultServeMux
				}
				h.ServeHTTP(w, r)

				return nil
			},
		})
	}

	s.router.Path("/{hostAndPort}/{profile}/").HandlerFunc(handler)
	s.router.Path("/{hostAndPort}/{profile}/{rest}").HandlerFunc(handler)
	return s
}

func main() {
	router := mux.NewRouter()

	router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
	router.PathPrefix("/pprof/").Handler(http.StripPrefix("/pprof", NewPProfServer().router))

	http.ListenAndServe(":8888", router)
}

// GoFlags implements the plugin.FlagSet interface.
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
