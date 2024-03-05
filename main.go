package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"os"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/xyzj/gopsu/config"
	"github.com/xyzj/gopsu/gocmd"
	"github.com/xyzj/gopsu/pathtool"
)

var (
	gover      = ""
	cover      = ""
	configfile = pathtool.JoinPathFromHere("go-mqttd.conf")
	conf       = config.NewConfig("")
	confile    = flag.String("config", "", "config file path, default is go-mqttd.conf")
)

func GetServerTLSConfig(certfile, keyfile, clientca string) (*tls.Config, error) {
	cliCrt, err := tls.LoadX509KeyPair(certfile, keyfile)
	if err != nil {
		return nil, err
	}
	tc := &tls.Config{
		ClientAuth: tls.NoClientCert,
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		},
		Certificates: []tls.Certificate{cliCrt},
	}
	if clientca != "" {
		caCrt, err := os.ReadFile(clientca)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		if pool.AppendCertsFromPEM(caCrt) {
			tc.ClientCAs = pool
			tc.ClientAuth = tls.RequireAndVerifyClientCert
		}
	}
	return tc, nil
}
func main() {
	svr := mqtt.New(nil)
	gocmd.DefaultProgram(&gocmd.Info{
		Ver:      "Core ver:\t" + cover + "\nGo ver:\t" + gover,
		Title:    "golang mqtt broker",
		Descript: "based on github.com/mochi-mqtt/server",
	}).AfterStop(func() {
		svr.Close()
	}).Execute()
	if *confile == "" {
		*confile = configfile
	}
	//  读取配置
	conf.FromFile(*confile)

	lconf := &listeners.Config{}
	tl, err := GetServerTLSConfig("localhost.pem", "localhost-key.pem", "")
	if err != nil {
		println(err.Error())
		lconf = nil
	} else {
		lconf.TLSConfig = tl
	}
	// 检查端口
	svr.AddHook(&auth.Hook{}, &auth.Options{
		Ledger: &auth.Ledger{
			Auth: auth.AuthRules{
				{Username: "arx7", Password: "arbalest", Allow: true},
				{Username: "YoRHa", Password: "no2typeB", Remote: "127.0.0.1", Allow: true},
			},
		}})
	tcp := listeners.NewTCP("mqtt-svr", ":"+conf.GetDefault(&config.Item{
		Key:     "mqtt_port",
		Value:   "1883",
		Comment: "mqtt服务端口",
	}).String(), lconf)
	err = svr.AddListener(tcp)
	if err != nil {
		println("MQTTd", ""+err.Error())
		return
	}
	web := listeners.NewHTTPStats("web", ":"+conf.GetDefault(&config.Item{
		Key:     "http_port",
		Value:   "1880",
		Comment: "mqtt服务状态查看端口",
	}).String(), lconf, svr.Info)
	err = svr.AddListener(web)
	if err != nil {
		println("HTTP", ""+err.Error())
		return
	}
	err = svr.Serve()
	if err != nil {
		println("MQTTd", ""+err.Error())
		return
	}
	conf.ToFile()
	select {}
}
