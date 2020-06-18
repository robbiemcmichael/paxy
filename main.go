package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/robbiemcmichael/paxy/internal"
	"github.com/robbiemcmichael/paxy/pkg/proxy"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	log.SetReportCaller(false)

	formatter := new(log.TextFormatter)
	formatter.FullTimestamp = true
	formatter.TimestampFormat = "15:04:05"
	log.SetFormatter(formatter)
}

func usage() {
	fmt.Printf("Usage: %s [options] pac_file ...\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	port := flag.Int("p", 8228, "The port on which the server listens")

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	pac := internal.PAC{
		File: flag.Arg(0),
	}

	if err := pac.Init(); err != nil {
		log.Fatalf("Failed to initialise PAC: %s", err)
	}

	paxy := &proxy.Proxy{
		Forward: pac.Evaluate,
	}

	if err := paxy.Init(); err != nil {
		log.Fatalf("Failed to initialise proxy: %s", err)
	}

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	log.Fatal(http.ListenAndServe(addr, paxy))
}
