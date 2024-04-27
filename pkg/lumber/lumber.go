package lumber

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net"
	"sync"
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
	cnf  Config
	l    *sync.Mutex
}

func (c *client) Send(batch []interface{}) error {
	c.l.Lock()
	defer c.l.Unlock()

	err := c.conn.Send(batch)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) ReDial() error {
	c.l.Lock()
	defer c.l.Unlock()

	err := c.Close()
	if err != nil {
		return err
	}

	newConn, err := dial(c.cnf)
	if err != nil {
		return err
	}

	c.conn = newConn

	return nil
}
func (c *client) Close() error {
	return c.conn.Close()
}

func NewClient(cnf Config) (*client, error) {
	cl, err := dial(cnf)
	if err != nil {
		log.Fatal(err)
	}
	return &client{
		cnf:  cnf,
		conn: cl,
		l:    &sync.Mutex{},
	}, nil
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
		log.Fatal("error in unmarshal winlog : ", err.Error())
	}
	dateNow := time.Now().Format("2006-01-02T15:04:05")
	res["@timestamp"] = dateNow
	return res
}

func dial(cnf Config) (*v2.Client, error) {
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
	return cl, nil
}
