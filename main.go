package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	tls_config "github.com/mrtdeh/testeps/pkg"
	"github.com/mrtdeh/testeps/pkg/lumber"
)

type JsonLogs struct {
	Name     string   `json:"name"`
	Port     int      `json:"port"`
	Protocol string   `json:"protocol"`
	Secure   bool     `json:"tls"`
	Logs     []string `json:"logs"`
}

type LogsMap map[string]JsonLogs

var (
	dest, sources *string
	ca, cert, key *string
	inifity       *bool
	delay         *int64

	logsMap   LogsMap = make(LogsMap)
	tlsConfig *tls.Config
	SendDelay time.Duration
)

func main() {
	err := loadSetting()
	if err != nil {
		log.Fatal(err)
	}

	dest = flag.String("c", "", "destination address")
	sources = flag.String("s", "", "sources")
	inifity = flag.Bool("i", false, "inifity mode")
	delay = flag.Int64("d", 0, "delay for each turn in inifity mode (miliseconds)")
	ca = flag.String("ca", "", "ca certificate")
	cert = flag.String("cert", "", "cert certificate")
	key = flag.String("key", "", "key certificate")
	flag.Parse()

	SendDelay = time.Duration(*delay) * time.Millisecond

	var incs []string
	if *sources != "" {
		incs = strings.Split(*sources, ",")
	}

	tlsConfig, err = tls_config.LoadTLSCredentials(tls_config.Config{
		CAPath:   *ca,
		CertPath: *cert,
		KeyPath:  *key,
	})
	if err != nil {
		// log.Println("tls config : ", err.Error())
	}

	var wg sync.WaitGroup
	if len(incs) > 0 {
		for _, i := range incs {
			if log, ok := logsMap[i]; ok {
				wg.Add(1)
				go func(i string) {
					defer wg.Done()
					send(*dest, log)
				}(i)
			}
		}
	} else {
		wg.Add(1)
		for k, v := range logsMap {
			go func(k string, v JsonLogs) {
				defer wg.Done()
				send(*dest, v)
			}(k, v)
		}
	}

	wg.Wait()
}

func sendBeatsLogs(ip string, mylog JsonLogs) {
	// Address
	addr := fmt.Sprintf("%s:%d", ip, mylog.Port)
	// Default setting of conenction
	lconf := lumber.Config{
		Addr:          addr,
		CompressLevel: 3,
		Timeout:       30 * time.Second,
		BatchSize:     1,
	}
	// Enable TLS if requested
	if mylog.Secure {
		if tlsConfig == nil {
			log.Fatalf("you want to use tls for %s but not specified certs/keys", mylog.Name)
		}
		lconf.TLSConfig = tlsConfig
	}
	// Create connection
	lc, err := lumber.NewClient(lconf)
	if err != nil {
		log.Fatalf("Failed to connect to Beat: %v", err)
	}
	defer lc.Close()

	for {
		for _, msg := range mylog.Logs {
			// Convert message to beat log
			m := lumber.M2(msg)
			// Overwrite timestamp field
			dateNow := time.Now().Format("2006-01-02T15:04:05")
			if _, ok := m.(map[string]interface{})["@timestamp"]; ok {
				m.(map[string]interface{})["@timestamp"] = dateNow
			} else {
				log.Fatal("@timestampe field not found in the json to overwrite")
			}
			// Send payload data
			payload := []interface{}{m}
			err := lc.Send(payload)
			if err != nil {
				log.Fatalf("Failed to send log to Beat: %v", err)
			}
		}
		log.Printf("send %d logs from datasource %s to server %s", len(mylog.Logs), mylog.Name, addr)
		time.Sleep(SendDelay)

		if !*inifity {
			break
		}
	}

}

func sendSyslogLogs(ip string, mylog JsonLogs) {
	var conn net.Conn
	var err error
	addr := fmt.Sprintf("%s:%d", ip, mylog.Port)

	if mylog.Secure {
		if tlsConfig == nil {
			log.Fatalf("you want to use tls for %s but not specified certs/keys", mylog.Name)
		}
		conn, err = tls.Dial(mylog.Protocol, addr, tlsConfig)
	} else {
		conn, err = net.Dial(mylog.Protocol, addr)
	}
	if err != nil {
		log.Fatalf("Failed to connect to tcp: %v", err)
	}
	defer conn.Close()

	for {
		for _, msg := range mylog.Logs {
			_, err := fmt.Fprintln(conn, msg)
			if err != nil {
				log.Fatalf("Failed to send log to Beat: %v", err)
			}
		}
		log.Printf("send %d logs from datasource %s to server %s", len(mylog.Logs), mylog.Name, addr)
		time.Sleep(SendDelay)

		if !*inifity {
			break
		}
	}
}

func load(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("%s not found : %s", path, err.Error())
	}
	lines := strings.Split(string(data), "\n")
	return lines
}

func send(ip string, log JsonLogs) {
	if log.Protocol == "tcp" || log.Protocol == "udp" {
		sendSyslogLogs(ip, log)
	} else if log.Protocol == "beats" {
		sendBeatsLogs(ip, log)
	}
}

func loadSetting() error {
	logsMap = LogsMap{}

	data, err := os.ReadFile("./sources.json")
	if err != nil {
		return fmt.Errorf("sources.json not exist an current directory!")
	}

	var j map[string]interface{}
	err = json.Unmarshal(data, &j)
	if err != nil {
		return err
	}

	if sources, ok := j["sources"]; ok {
		if ss, ok := sources.([]interface{}); ok {
			for _, s := range ss {
				tmp := s.(map[string]interface{})
				logsMap[tmp["name"].(string)] = JsonLogs{
					Name:     tmp["name"].(string),
					Port:     int(tmp["port"].(float64)),
					Protocol: tmp["proto"].(string),
					Secure:   tmp["tls"].(bool),
					Logs:     load(tmp["path"].(string)),
				}
			}
		} else {
			return fmt.Errorf("parse failed : sources must be a array")
		}
	} else {
		return fmt.Errorf("parse failed : sources not found")
	}

	return nil

}
