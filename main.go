package main

import (
	"crypto/tls"
	"flag"
	"log"
	"strings"
	"time"

	"github.com/mrtdeh/testeps/pkg/core"
)

var (
	dest, sources *string
	inifity       *bool
	delay         *int64
	threads       *int

	showSources, editSources *bool

	tlsConfig *tls.Config
	SendDelay time.Duration
)

func main() {
	err := core.LoadSetting()
	if err != nil {
		log.Fatal(err)
	}

	// switchs
	dest = flag.String("c", "", "destination address")
	sources = flag.String("s", "", "sources")
	inifity = flag.Bool("i", false, "inifity mode")
	delay = flag.Int64("d", 0, "delay for each turn in inifity mode (miliseconds)")
	threads = flag.Int("t", 1, "threads count")
	// commands
	showSources = flag.Bool("show", false, "show sources list")
	editSources = flag.Bool("edit", false, "edit sources")
	flag.Parse()

	if *showSources {
		core.PrintSources()
		return
	}

	if *editSources {
		core.EditSources()
		return
	}

	var incs []string
	if *sources != "" {
		incs = strings.Split(*sources, ",")
	}
	core.Run(core.Config{
		DestinationIp: *dest,
		Sources:       incs,
		SendDelay:     time.Duration(*delay) * time.Millisecond,
		Inifity:       *inifity,
		ThreadsCount:  *threads,
	})
}
