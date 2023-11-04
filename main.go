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
	ca, cert, key *string
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

	dest = flag.String("c", "", "destination address")
	sources = flag.String("s", "", "sources")
	inifity = flag.Bool("i", false, "inifity mode")
	delay = flag.Int64("d", 0, "delay for each turn in inifity mode (miliseconds)")
	threads = flag.Int("t", 1, "threads count")

	showSources = flag.Bool("show-sources", false, "show sources list")
	editSources = flag.Bool("edit-sources", false, "edit sources")

	ca = flag.String("ca", "", "ca certificate")
	cert = flag.String("cert", "", "cert certificate")
	key = flag.String("key", "", "key certificate")
	flag.Parse()

	if *showSources {
		core.PrintSources()
		return
	}

	if *editSources {
		core.EditSources()
		return
	}
	SendDelay = time.Duration(*delay) * time.Millisecond

	var incs []string
	if *sources != "" {
		incs = strings.Split(*sources, ",")
	}
	core.Run(core.Config{
		Sources:   incs,
		SendDelay: SendDelay,
		// TLSConfig:     tlsConfig,
		Inifity:       *inifity,
		ThreadsCount:  *threads,
		DestinationIp: *dest,
	})
}
