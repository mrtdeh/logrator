package core

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/mrtdeh/testeps/pkg/lumber"
	"github.com/rodaine/table"
)

type JsonLogs struct {
	Name     string   `json:"name"`
	Port     int      `json:"port"`
	Protocol string   `json:"protocol"`
	Secure   bool     `json:"tls"`
	Logs     []string `json:"logs"`
}

type LogsMap map[string]JsonLogs

type Config struct {
	Sources       []string
	SendDelay     time.Duration
	TLSConfig     *tls.Config
	Inifity       bool
	ThreadsCount  int
	DestinationIp string
}

var (
	logsMap LogsMap = make(LogsMap)
)

func Run(cnf Config) {
	incs := cnf.Sources

	var wg sync.WaitGroup
	if len(incs) == 0 {
		for k := range logsMap {
			incs = append(incs, k)
		}
	}

	for _, i := range incs {
		if log, ok := logsMap[i]; ok {
			wg.Add(1)
			go func(i string) {
				// var count int
				defer wg.Done()
				for j := 0; j <= cnf.ThreadsCount; j++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						cnf.send(log)
					}()
				}
			}(i)
		}
	}

	wg.Wait()
}

func (c *Config) sendBeatsLogs(mylog JsonLogs) {
	// Address
	addr := fmt.Sprintf("%s:%d", c.DestinationIp, mylog.Port)
	// Default setting of conenction
	lconf := lumber.Config{
		Addr:          addr,
		CompressLevel: 3,
		Timeout:       30 * time.Second,
		BatchSize:     1,
	}
	// Enable TLS if requested
	if mylog.Secure {
		if c.TLSConfig == nil {
			log.Fatalf("you want to use tls for %s but not specified certs/keys", mylog.Name)
		}
		lconf.TLSConfig = c.TLSConfig
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
		time.Sleep(c.SendDelay)

		if !*&c.Inifity {
			break
		}
	}

}

func (c *Config) sendSyslogLogs(mylog JsonLogs) {
	var conn net.Conn
	var err error
	addr := fmt.Sprintf("%s:%d", c.DestinationIp, mylog.Port)

	if mylog.Secure {
		if c.TLSConfig == nil {
			log.Fatalf("you want to use tls for %s but not specified certs/keys", mylog.Name)
		}
		conn, err = tls.Dial(mylog.Protocol, addr, c.TLSConfig)
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
		time.Sleep(c.SendDelay)

		if !*&c.Inifity {
			break
		}
	}
}

func (c *Config) send(log JsonLogs) {
	if log.Protocol == "tcp" || log.Protocol == "udp" {
		c.sendSyslogLogs(log)
	} else if log.Protocol == "beats" {
		c.sendBeatsLogs(log)
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

func LoadSetting() error {
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

func PrintSources() {
	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("Name", "Protocol", "Port", "TLS", "LogsCount")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	keys := make([]string, 0, len(logsMap))

	for k := range logsMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := logsMap[k]
		s := "No"
		if v.Secure {
			s = "Yes"
		}
		tbl.AddRow(v.Name, v.Protocol, v.Port, s, len(v.Logs))
	}

	tbl.Print()
}
