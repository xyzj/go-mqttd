package main

import (
	"flag"
	"go-mqttd/server"

	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/config"
	"github.com/xyzj/gopsu/crypto"
	"github.com/xyzj/gopsu/gocmd"
	"github.com/xyzj/gopsu/pathtool"
)

var (
	gover       = ""
	cover       = ""
	confname    = "go-mqttd.conf"
	confile     = flag.String("config", "", "config file path, default is "+confname)
	authfile    = flag.String("auth", "", "auth file path")
	disableAuth = flag.Bool("disable-auth", false, "disable auth check, ignore -auth")
)

type svrOpt struct {
	conf   *config.File
	mqtt   int    // mqtt port
	tls    int    // mqtt+tls port
	web    int    // http status port
	ws     int    // websocket port
	cert   string // tls cert file path
	key    string // tls key file path
	rootca string
}

func loadConf(configfile string) *svrOpt {
	conf := config.NewConfig("")
	//  load config
	conf.FromFile(configfile)
	o := &svrOpt{}
	o.tls = conf.GetDefault(&config.Item{
		Key:     "port_tls",
		Value:   "1881",
		Comment: "mqtt+tls port",
	}).TryInt()

	o.mqtt = conf.GetDefault(&config.Item{
		Key:     "port_mqtt",
		Value:   "1883",
		Comment: "mqtt port",
	}).TryInt()
	o.web = conf.GetDefault(&config.Item{
		Key:     "port_web",
		Value:   "1880",
		Comment: "http status port",
	}).TryInt()
	o.ws = conf.GetDefault(&config.Item{
		Key:     "port_ws",
		Value:   "",
		Comment: "websocket port, default: 1882",
	}).TryInt()
	o.cert = conf.GetDefault(&config.Item{
		Key:     "tls_cert_file",
		Value:   "cert.ec.pem",
		Comment: "tls cert file path",
	}).String()
	o.key = conf.GetDefault(&config.Item{
		Key:     "tls_key_file",
		Value:   "cert-key.ec.pem",
		Comment: "tls key file path",
	}).String()
	o.rootca = conf.GetDefault(&config.Item{
		Key:     "tls_ca_file",
		Value:   "",
		Comment: "tls root ca file path",
	}).String()
	o.conf = conf
	// save config
	conf.ToFile()
	return o
}

func main() {
	var svr *server.MqttServer
	gocmd.DefaultProgram(&gocmd.Info{
		Ver:      "Core ver: " + cover + "\nGo ver:   " + gover,
		Title:    "golang mqtt broker",
		Descript: "based on mochi-mqtt, support MQTT v3.11 and MQTT v5.0",
	}).AddCommand(&gocmd.Command{
		Name:     "genecc",
		Descript: "generate ECC certificate files",
		RunWithExitCode: func(pi *gocmd.ProcInfo) int {
			c := crypto.NewECC()
			ips, _, err := gopsu.GlobalIPs()
			if err != nil {
				ips = []string{"127.0.0.1"}
			}
			local := false
			for _, v := range ips {
				if v == "127.0.0.1" {
					local = true
				}
			}
			if !local {
				ips = append(ips, "127.0.0.1")
			}
			if err := c.CreateCert(&crypto.CertOpt{
				DNS: []string{"localhost"},
				IP:  ips,
			}); err != nil {
				println(err.Error())
				return 1
			}
			println("done.")
			return 0
		},
	}).AfterStop(func() {
		svr.Stop()
	}).Execute()

	if *confile == "" {
		*confile = pathtool.JoinPathFromHere(confname)
	}
	o := loadConf(*confile)

	svr = server.NewServer(&server.Opt{
		PortTLS:     o.tls,
		PortWeb:     o.web,
		PortWS:      o.ws,
		PortMqtt:    o.mqtt,
		Cert:        o.cert,
		Key:         o.key,
		RootCA:      o.rootca,
		DisableAuth: *disableAuth,
		Authfile:    *authfile,
	})
	svr.Run()
}
