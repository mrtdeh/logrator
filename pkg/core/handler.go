package core

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/goccy/go-yaml"
	"github.com/mrtdeh/testeps/pkg/lumber"
	"github.com/mrtdeh/testeps/pkg/tls_config"
	"github.com/rodaine/table"
)

type JsonLogs struct {
	Name     string `yaml:"name"`
	Port     int    `yaml:"port"`
	Protocol string `yaml:"proto"`
	Secure   bool   `yaml:"ssl"`
	Path     string `yaml:"path"`

	CA   string `yaml:"ssl_ca"`
	Cert string `yaml:"ssl_cert"`
	Key  string `yaml:"ssl_key"`

	logs []string
}

type Sources struct {
	Default struct {
		CA   string `yaml:"ssl_ca"`
		Cert string `yaml:"ssl_cert"`
		Key  string `yaml:"ssl_key"`
	} `yaml:"default"`
	Sources []JsonLogs `yaml:"sources"`
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

func LoadSetting() error {
	logsMap = LogsMap{}

	data, err := os.ReadFile("./sources.yaml")
	if err != nil {
		return fmt.Errorf("sources.yaml not exist an current directory!")
	}

	var j Sources
	// err = json.Unmarshal(data, &j)
	err = yaml.Unmarshal(data, &j)
	if err != nil {
		return err
	}

	if j.Sources != nil {
		for _, s := range j.Sources {
			s.logs = load(s.Path)
			if s.CA == "" {
				s.CA = j.Default.CA
			}
			if s.Cert == "" {
				s.Cert = j.Default.Cert
			}
			if s.Key == "" {
				s.Key = j.Default.Key
			}

			logsMap[s.Name] = s
		}
	} else {
		return fmt.Errorf("parse failed : sources not found")
	}

	return nil

}

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
				for j := 0; j < cnf.ThreadsCount; j++ {
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
		lconf.TLSConfig = c.TLSConfig
	}
	// Create connection
	lc, err := lumber.NewClient(lconf)
	if err != nil {
		log.Fatalf("Failed to connect to Beat: %v", err)
	}
	defer lc.Close()

	for {
		for _, msg := range mylog.logs {
			// Convert message to beat log
			payload := []interface{}{lumber.M2(msg)}
			// Send payload data
			err := lc.Send(payload)
			if err != nil {
				log.Fatalf("Failed to send log to Beat: %v", err)
			}
		}
		log.Printf("send %d logs from datasource %s to server %s", len(mylog.logs), mylog.Name, addr)
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
		conn, err = tls.Dial(mylog.Protocol, addr, c.TLSConfig)
	} else {
		conn, err = net.Dial(mylog.Protocol, addr)
	}
	if err != nil {
		log.Fatalf("Failed to connect to tcp: %v", err)
	}
	defer conn.Close()

	for {
		for _, msg := range mylog.logs {
			_, err := fmt.Fprintln(conn, msg)
			if err != nil {
				log.Fatalf("Failed to send log to Beat: %v", err)
			}
		}
		log.Printf("send %d logs from datasource %s to server %s", len(mylog.logs), mylog.Name, addr)
		time.Sleep(c.SendDelay)

		if !*&c.Inifity {
			break
		}
	}
}

func (c *Config) send(mylog JsonLogs) {
	var tlsConfig *tls.Config
	var err error

	if mylog.Secure {
		tlsConfig, err = tls_config.LoadTLSCredentials(tls_config.Config{
			CAPath:   mylog.CA,
			CertPath: mylog.Cert,
			KeyPath:  mylog.Key,
		})
		if err != nil {
			log.Fatal("tls config : ", err.Error())
		}
		c.TLSConfig = tlsConfig
	}

	if mylog.Protocol == "tcp" || mylog.Protocol == "udp" {
		c.sendSyslogLogs(mylog)
	} else if mylog.Protocol == "beats" {
		c.sendBeatsLogs(mylog)
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
		tbl.AddRow(v.Name, v.Protocol, v.Port, s, len(v.logs))
	}

	tbl.Print()
}

func EditSources() error {
	fpath := "/usr/share/logrator/sources.yaml"
	f, err := os.Open(fpath)
	if err != nil {
		log.Fatal(err)
	}
	f.Close()

	cmd := exec.Command("nano", fpath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	err = cmd.Wait()
	if err != nil {
		log.Printf("Error while editing. Error: %v\n", err)
	} else {
		log.Printf("Successfully edited.")
	}

	return nil

}
