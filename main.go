package main

import (
	"flag"
	listener "go-mqttd/listener"
	"os"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/xyzj/gopsu"
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
	disableAuth = flag.Bool("disable-auth", false, "disable auth check, ignore -auth")
)

type opt struct {
	mqtt string // mqtt port
	tls  string // mqtt+tls port
	web  string // http status port
	ws   string // websocket port
	cert string // tls cert file path
	key  string // tls key file path
}

func loadConf() *opt {
	o := &opt{}
	if *confile != "" {
		configfile = *confile
	}
	//  load config
	conf.FromFile(configfile)
	o.tls = conf.GetDefault(&config.Item{
		Key:     "mqtt_tls_port",
		Value:   "1882",
		Comment: "mqtt+tls port",
	}).String()

	o.mqtt = conf.GetDefault(&config.Item{
		Key:     "mqtt_port",
		Value:   "1883",
		Comment: "mqtt port",
	}).String()
	o.web = conf.GetDefault(&config.Item{
		Key:     "mqtt_web",
		Value:   "1880",
		Comment: "http status port",
	}).String()
	o.ws = conf.GetDefault(&config.Item{
		Key:     "mqtt_ws",
		Value:   "",
		Comment: "websocket port, default: 1881",
	}).String()
	o.cert = conf.GetDefault(&config.Item{
		Key:     "mqtt_tls_cert",
		Value:   "localhost.pem",
		Comment: "tls cert file path",
	}).String()
	o.key = conf.GetDefault(&config.Item{
		Key:     "mqtt_tls_key",
		Value:   "localhost-key.pem",
		Comment: "tls key file path",
	}).String()
	// save config
	if *confile != "" {
		conf.ToFile()
	}
	return o
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
	// read buffer size env
	size := gopsu.String2Int(os.Getenv("MQTT_CLIENT_BUFFER_SIZE"), 10)
	if size == 0 {
		size = 4
	}
	// a new svr
	svr := mqtt.New(&mqtt.Options{
		InlineClient:             false,
		ClientNetWriteBufferSize: 1024 * size,
		ClientNetReadBufferSize:  1024 * size,
	})
	gocmd.DefaultProgram(&gocmd.Info{
		Ver:      "Core ver: " + cover + "\nGo ver:   " + gover,
		Title:    "golang mqtt broker",
		Descript: "based on mochi-mqtt, support MQTT v3.11 and MQTT v5.0",
	}).AfterStop(func() {
		svr.Close()
	}).Execute()
	// load config
	o := loadConf()
	// check tls files
	tl, err := gopsu.GetServerTLSConfig(o.cert, o.key, "")
	if err != nil {
		o.tls = ""
		svr.Log.Warn(err.Error())
	}
	// set auth
	if !*disableAuth {
		au := fromAuthFile()
		// add two admin accounts
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
	// mqtt tls service
	if o.tls != "" {
		err = svr.AddListener(listeners.NewTCP(listeners.Config{
			ID:        "tls",
			Address:   ":" + o.tls,
			TLSConfig: tl,
		}))
		if err != nil {
			svr.Log.Error("MQTT+TLS service error: " + err.Error())
			return
		}
	}
	// mqtt service
	if o.mqtt != "" {
		err = svr.AddListener(listeners.NewTCP(listeners.Config{
			ID:        "mqtt",
			Address:   ":" + o.mqtt,
			TLSConfig: nil,
		}))
		if err != nil {
			svr.Log.Error("MQTT service error: " + err.Error())
			return
		}
	}
	// http status service
	if o.web != "" {
		err = svr.AddListener(listener.NewHTTPStats("web", ":"+o.web, &listeners.Config{}, svr.Info, svr.Clients))
		if err != nil {
			svr.Log.Error("HTTP service error: " + err.Error())
			return
		}
	}
	// websocket service
	if o.ws != "" {
		err = svr.AddListener(listeners.NewWebsocket(listeners.Config{
			ID:        "ws",
			Address:   ":" + o.ws,
			TLSConfig: nil,
		}))
		if err != nil {
			svr.Log.Error("WS service error: " + err.Error())
			return
		}
	}
	// start serve
	err = svr.Serve()
	if err != nil {
		svr.Log.Error("serve error: " + err.Error())
		return
	}
	select {}
}
