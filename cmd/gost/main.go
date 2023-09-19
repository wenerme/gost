package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/go-gost/x/registry"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/go-gost/core/logger"
	xlogger "github.com/go-gost/x/logger"
	"github.com/judwhite/go-svc"
)

var (
	cfgFile      string
	outputFormat string
	services     stringList
	nodes        stringList
	debug        bool
	apiAddr      string
	metricsAddr  string
)

func init() {
	args := strings.Join(os.Args[1:], "  ")

	if strings.Contains(args, " -- ") {
		var (
			wg  sync.WaitGroup
			ret int
		)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for wid, wargs := range strings.Split(" "+args+" ", " -- ") {
			wg.Add(1)
			go func(wid int, wargs string) {
				defer wg.Done()
				defer cancel()
				worker(wid, strings.Split(wargs, "  "), &ctx, &ret)
			}(wid, strings.TrimSpace(wargs))
		}

		wg.Wait()

		os.Exit(ret)
	}
}

func worker(id int, args []string, ctx *context.Context, ret *int) {
	cmd := exec.CommandContext(*ctx, os.Args[0], args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("_GOST_ID=%d", id))

	cmd.Run()
	if cmd.ProcessState.Exited() {
		*ret = cmd.ProcessState.ExitCode()
	}
}

func init() {
	var printVersion bool

	flag.Var(&services, "L", "service list")
	flag.Var(&nodes, "F", "chain node list")
	flag.StringVar(&cfgFile, "C", "", "configuration file")
	flag.BoolVar(&printVersion, "V", false, "print version")
	flag.StringVar(&outputFormat, "O", "", "output format, one of yaml|json format")
	flag.BoolVar(&debug, "D", false, "debug mode")
	flag.StringVar(&apiAddr, "api", "", "api service address")
	flag.StringVar(&metricsAddr, "metrics", "", "metrics service address")
	flag.Parse()

	if printVersion {
		fmt.Fprintf(os.Stdout, "gost %s (%s %s/%s)\n",
			version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	logger.SetDefault(xlogger.NewLogger())
}

func main() {
	if ctlrAddr != "" {
		ctlrAddr = strings.TrimSuffix(ctlrAddr, "/")
		registry.AutherRegistry().Register("controller/auther", &auth{})
	}

	p := &program{}

	if err := svc.Run(p); err != nil {
		log.Fatal(err)
	}
}

var ctlrAddr = os.Getenv("GOST_CONTROLLER_ADDR")

type auth struct {
}

func (a *auth) Authenticate(ctx context.Context, user, password string) bool {
	res, err := http.Post(ctlrAddr+"/auth", "application/json", bytes.NewReader(MustMarshal(map[string]string{
		"username": user,
		"password": password,
	})))
	if err != nil {
		return false
	}
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return true
	}
	return false
}

func MustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
