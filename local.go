package cracker

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	g "github.com/golang/glog"
	"gopkg.in/bufio.v1"
)

var hc = &http.Client{}

func Init(cert string) {
	if f, err := os.Stat(cert); err == nil && !f.IsDir() {
		var CAPOOL *x509.CertPool
		CAPOOL, err := x509.SystemCertPool()
		if err != nil {
			g.Warning(err)
			CAPOOL = x509.NewCertPool()
		}
		serverCert, err := ioutil.ReadFile(cert)
		if err != nil {
			g.Errorf("read cert.pem err:%s ", err)
			return
		}
		CAPOOL.AppendCertsFromPEM(serverCert)
		tp := hc.Transport.(*http.Transport)
		config := &tls.Config{RootCAs: CAPOOL}
		tp.TLSClientConfig = config
		g.Infof("load %s success ... ", cert)
	} else if err != nil {
		g.Error(err)
	} else {
		g.Errorf("%s is a dir", cert)
	}
}

type localProxyConn struct {
	uuid     string
	server   string
	secret   string
	source   io.ReadCloser
	close    chan bool
	interval time.Duration
	dst      io.WriteCloser
}

func (c *localProxyConn) gen_sign(req *http.Request) {

	ts := fmt.Sprintf("%d", time.Now().Unix())
	req.Header.Set("UUID", c.uuid)
	req.Header.Set("timestamp", ts)
	req.Header.Set("sign", GenHMACSHA1(c.secret, ts))
}

func (c *localProxyConn) chunkPush(data []byte, typ string) error {
	if c.dst != nil {
		_, err := c.dst.Write(data)
		return err
	}
	wr, ww := io.Pipe()
	req, _ := http.NewRequest("POST", c.server+PUSH, wr)
	req.Header.Set("TYP", typ)
	req.Header.Set("Transfer-Encoding", "chunked")
	c.gen_sign(req)
	req.Header.Set("Content-Type", "image/jpeg")
	go func() (err error) {
		defer wr.Close()
		defer ww.Close()
		res, err := hc.Do(req)
		if err != nil {
			return
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		switch res.StatusCode {
		case HeadOK:
			return nil
		default:
			return errors.New(fmt.Sprintf("status code is %d, body is: %s", res.StatusCode, string(body)))
		}
		return nil
	}()

	c.dst = ww
	_, err := c.dst.Write(data)
	return err
}

func (c *localProxyConn) push(data []byte, typ string) error {
	buf := bufio.NewBuffer(data)
	req, _ := http.NewRequest("POST", c.server+PUSH, buf)
	req.Header.Set("TYP", typ)
	c.gen_sign(req)
	req.ContentLength = int64(len(data))
	req.Header.Set("Content-Type", "image/jpeg")
	res, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	switch res.StatusCode {
	case HeadOK:
		return nil
	default:
		return errors.New(fmt.Sprintf("status code is %d, body is: %s", res.StatusCode, string(body)))
	}
}

func (c *localProxyConn) connect(dstHost, dstPort string) (uuid string, err error) {
	req, _ := http.NewRequest("GET", c.server+CONNECT, nil)
	c.gen_sign(req)
	req.Header.Set("DSTHOST", dstHost)
	req.Header.Set("DSTPORT", dstPort)
	cxt, cancel := context.WithTimeout(context.Background(), time.Second*timeout)
	defer cancel()
	req.WithContext(cxt)
	res, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	body, _ := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != HeadOK {
		return "", errors.New(fmt.Sprintf("status code is %d, body is:%s", res.StatusCode, string(body)))
	}
	return string(body), err

}

func (c *localProxyConn) pull() error {

	req, _ := http.NewRequest("GET", c.server+PULL, nil)
	req.Header.Set("Interval", fmt.Sprintf("%d", c.interval))
	c.gen_sign(req)
	if c.interval > 0 {
		cxt, cancel := context.WithTimeout(context.Background(), time.Second*timeout)
		defer cancel()
		req.WithContext(cxt)

	}
	res, err := hc.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != HeadOK {
		body, _ := ioutil.ReadAll(res.Body)
		res.Body.Close()
		return errors.New(fmt.Sprintf("status code is %d, body is %s", res.StatusCode, string(body)))
	}
	c.source = res.Body
	return nil
}

func (c *localProxyConn) Read(b []byte) (n int, err error) {

	if c.source == nil {
		if c.interval > 0 {
			if err = c.pull(); err != nil {
				return
			}
		} else {
			return 0, errors.New("pull http connection is not ready")
		}
	}
	n, err = c.source.Read(b)
	if err != nil {
		c.source.Close()
		c.source = nil
	}
	if err == io.EOF && c.interval > 0 {
		err = nil
	}
	return
}

func (c *localProxyConn) Write(b []byte) (int, error) {

	var err error
	if c.interval > 0 {
		err = c.push(b, DATA_TYP)
	} else {
		//err = c.push(b, DATA_TYP)
		err = c.chunkPush(b, DATA_TYP)
	}
	if err != nil {
		g.V(LDEBUG).Infof("push: %v", err)
		return 0, err
	}

	return len(b), nil
}

func (c *localProxyConn) alive() {
	for {
		select {
		case <-c.close:
			return
		case <-time.After(time.Second * heartTTL / 2):
			if err := c.push([]byte("alive"), HEART_TYP); err != nil {
				return
			}
		}
	}
}

func (c *localProxyConn) quit() error {
	return c.push([]byte("quit"), QUIT_TYP)
}

func (c *localProxyConn) Close() error {
	close(c.close)
	return c.quit()
}
