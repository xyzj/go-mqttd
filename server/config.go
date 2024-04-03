package server

import "github.com/xyzj/gopsu/config"

func loadConf(configfile string) *svrOpt {
	conf := config.NewConfig("")
	//  load config
	conf.FromFile(configfile)
	o := &svrOpt{}
	o.tls = conf.GetDefault(&config.Item{
		Key:     "port_tls",
		Value:   "1881",
		Comment: "mqtt+tls port",
	}).String()

	o.mqtt = conf.GetDefault(&config.Item{
		Key:     "port_mqtt",
		Value:   "1883",
		Comment: "mqtt port",
	}).String()
	o.web = conf.GetDefault(&config.Item{
		Key:     "port_web",
		Value:   "1880",
		Comment: "http status port",
	}).String()
	o.ws = conf.GetDefault(&config.Item{
		Key:     "port_ws",
		Value:   "",
		Comment: "websocket port, default: 1882",
	}).String()
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
	o.conf = conf
	// save config
	// if *confile != "" {
	// 	conf.ToFile()
	// }
	return o
}
