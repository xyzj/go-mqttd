package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	listener "go-mqttd/listener"
	"os"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/xyzj/gopsu/config"
	"github.com/xyzj/gopsu/gocmd"
	"github.com/xyzj/gopsu/pathtool"
	"gopkg.in/yaml.v3"
)

var (
	gover       = ""
	cover       = ""
	confname    = "stmq.conf"
	configfile  = pathtool.JoinPathFromHere(confname)
	conf        = config.NewConfig("")
	confile     = flag.String("config", "", "config file path, default is "+confname)
	authfile    = flag.String("auth", "", "auth file path")
	disableAuth = flag.Bool("disable-auth", false, "disable auth file, ignore -auth")
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

/*
	Filters Access :

0-Deny      				// user cannot access the topic
1-ReadOnly                // user can only subscribe to the topic
2-WriteOnly               // user can only publish to the topic
3-ReadWrite               // user can both publish and subscribe to the topic
*/
func fromAuthFile() *auth.Ledger {
	ac := &auth.Ledger{}
	if *authfile == "" {
		return ac
	}
	b, err := os.ReadFile(*authfile)
	if err != nil {
		createAuthFile(*authfile)
		return ac
	}

	err = yaml.Unmarshal(b, &ac)
	if err != nil {
		createAuthFile(*authfile)
		return &auth.Ledger{}
	}
	return ac
}
func createAuthFile(filename string) {
	b, err := yaml.Marshal(&auth.Ledger{
		Users: map[string]auth.UserRule{
			"mqttdevices": {
				Username: "mqttdevices",
				Password: "fallguys",
				ACL: auth.Filters{
					"down/#": auth.ReadOnly,
					"up/#":   auth.WriteOnly,
				}},
			"lostjudgment": {
				Username: "lostjudgment",
				Password: "yagami",
				ACL: auth.Filters{
					"deny/#": auth.Deny,
					"rw/#":   auth.ReadWrite,
				},
			},
		},
	})
	if err != nil {
		println(err.Error())
		return
	}
	err = os.WriteFile(filename, b, 0664)
	if err != nil {
		println(err.Error())
	}
}
func main() {
	svr := mqtt.New(nil)
	gocmd.DefaultProgram(&gocmd.Info{
		Ver:      "Core ver: " + cover + "\nGo ver:   " + gover,
		Title:    "golang mqtt broker",
		Descript: "",
	}).AfterStop(func() {
		svr.Close()
	}).Execute()

	if *confile != "" {
		configfile = *confile
	}
	//  读取配置
	conf.FromFile(configfile)
	tl, err := GetServerTLSConfig(
		conf.GetDefault(&config.Item{
			Key:     "cert_file",
			Value:   "localhost.pem",
			Comment: "cert file",
		}).String(),
		conf.GetDefault(&config.Item{
			Key:     "cert_key",
			Value:   "localhost-key.pem",
			Comment: "key file",
		}).String(),
		"")
	var mqtttls string
	if err != nil {
		println(err.Error())
	} else {
		mqtttls = ":" + conf.GetDefault(&config.Item{
			Key:     "port_mqtt_tls",
			Value:   "1881",
			Comment: "mqtt服务tls端口",
		}).String()
	}
	mqttport := ":" + conf.GetDefault(&config.Item{
		Key:     "port_mqtt",
		Value:   "1883",
		Comment: "mqtt服务端口",
	}).String()
	webport := ":" + conf.GetDefault(&config.Item{
		Key:     "port_web",
		Value:   "1880",
		Comment: "mqtt服务状态查看端口",
	}).String()
	// 保存配置
	if *confile != "" {
		conf.ToFile()
	}
	// 设置权限
	if !*disableAuth {
		au := fromAuthFile()
		// 追加内置账户
		au.Auth = append(au.Auth,
			auth.AuthRule{Username: "arx7", Password: "arbalest", Allow: true},
			auth.AuthRule{Username: "YoRHa", Password: "no2typeB", Remote: "127.0.0.1", Allow: true},
		)
		svr.AddHook(&auth.Hook{}, &auth.Options{
			Ledger: au,
		})
	} else {
		svr.AddHook(&auth.AllowHook{}, nil)
	}
	// mqtt tls端口
	if mqtttls != "" {
		err = svr.AddListener(listeners.NewTCP("mqtt-tls", mqtttls, &listeners.Config{
			TLSConfig: tl,
		}))
		if err != nil {
			println("MQTTd", ""+err.Error())
			return
		}
	}
	// // mqtt端口
	err = svr.AddListener(listeners.NewTCP("mqtt-svr", mqttport, &listeners.Config{}))
	if err != nil {
		println("MQTTd", ""+err.Error())
		return
	}
	// 监听网络状态
	err = svr.AddListener(listener.NewHTTPStats("web", webport, &listeners.Config{}, svr.Clients))
	if err != nil {
		println("HTTP", ""+err.Error())
		return
	}
	// 启动服务
	err = svr.Serve()
	if err != nil {
		println("MQTTd", ""+err.Error())
		return
	}
	select {}
}
