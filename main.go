package main

import (
	"crypto/tls"
	"flag"
	"log"
	"strings"
	"time"

	"github.com/mrtdeh/testeps/pkg/core"
	tls_config "github.com/mrtdeh/testeps/pkg/tls_config"
)

var (
	dest, sources *string
	ca, cert, key *string
	inifity       *bool
	delay         *int64
	threads       *int

	showSources *bool

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

	ca = flag.String("ca", "", "ca certificate")
	cert = flag.String("cert", "", "cert certificate")
	key = flag.String("key", "", "key certificate")
	flag.Parse()

	if *showSources {
		core.PrintSources()
		return
	}
	SendDelay = time.Duration(*delay) * time.Millisecond
	tlsConfig, err = tls_config.LoadTLSCredentials(tls_config.Config{
		CAPath:   *ca,
		CertPath: *cert,
		KeyPath:  *key,
	})
	if err != nil {
		// log.Println("tls config : ", err.Error())
	}

	var incs []string
	if *sources != "" {
		incs = strings.Split(*sources, ",")
	}
	core.Run(core.Config{
		Sources:       incs,
		SendDelay:     SendDelay,
		TLSConfig:     tlsConfig,
		Inifity:       *inifity,
		ThreadsCount:  *threads,
		DestinationIp: *dest,
	})
}
