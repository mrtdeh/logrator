package lumber

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net"
	"time"

	v2 "github.com/elastic/go-lumber/client/v2"
)

type Config struct {
	Addr          string
	CompressLevel int // Compression level 0-9
	Timeout       time.Duration
	BatchSize     int
	TLSConfig     *tls.Config
}

type client struct {
	conn *v2.Client
}

func (c *client) Send(batch []interface{}) error {
	err := c.conn.Send(batch)
	if err != nil {
		return err
	}
	return nil
}
func (c *client) Close() error {
	return c.conn.Close()
}

func NewClient(cnf Config) (*client, error) {
	cl, err := v2.DialWith(func(network, address string) (net.Conn, error) {
		if cnf.TLSConfig != nil {
			return tls.Dial(network, address, cnf.TLSConfig)
		}
		return net.Dial(network, address)
	}, cnf.Addr, v2.CompressionLevel(cnf.CompressLevel),
		v2.Timeout(cnf.Timeout),
	)
	if err != nil {
		return nil, err
	}

	return &client{cl}, nil
}

func M(msg string) interface{} {

	return map[string]interface{}{
		"@timestamp": time.Now(),
		"type":       "filebeat",
		"message":    msg,
		"offset":     1000,
	}
}

func M2(body string) interface{} {

	var res map[string]interface{}
	err := json.Unmarshal([]byte(body), &res)
	if err != nil {
		log.Fatal(err)
	}

	dateNow := time.Now().Format("2006-01-02T15:04:05")
	// if _, ok := res["@timestamp"]; ok {
	res["@timestamp"] = dateNow
	// } else {
	// 	log.Fatal("@timestampe field not found in the json to overwrite")
	// }

	return res
}
