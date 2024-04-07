package server

import (
	"go-mqttd/listener"
	"os"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/config"
)

// Opt server option
type Opt struct {
	// Confile net config file path
	Confile string
	// Authfile auth config file path
	Authfile string
	// DisableAuth clients do not need username and password
	DisableAuth bool
	// InsideJob enable or disable inline client
	InsideJob bool
}

// MqttServer a new mqtt server
type MqttServer struct {
	svr  *mqtt.Server
	opt  *Opt
	conf *svrOpt
}
type svrOpt struct {
	conf   *config.File
	mqtt   string // mqtt port
	tls    string // mqtt+tls port
	web    string // http status port
	ws     string // websocket port
	cert   string // tls cert file path
	key    string // tls key file path
	rootca string
}

// NewServer make a new server
func NewServer(opt *Opt) *MqttServer {
	// read buffer size env
	size := gopsu.String2Int(os.Getenv("MQTT_CLIENT_BUFFER_SIZE"), 10)
	if size == 0 {
		size = 4
	}
	// a new svr
	svr := mqtt.New(&mqtt.Options{
		InlineClient:             opt.InsideJob,
		ClientNetWriteBufferSize: 1024 * size,
		ClientNetReadBufferSize:  1024 * size,
	})
	// load config
	o := loadConf(opt.Confile)
	return &MqttServer{
		svr:  svr,
		opt:  opt,
		conf: o,
	}
}

// SaveConfig save server config to a file
func (m *MqttServer) SaveConfig() error {
	return m.conf.conf.ToFile()
}

// Stop close server
func (m *MqttServer) Stop() {
	m.svr.Close()
}

// Run start server and wait
func (m *MqttServer) Run() {
	if m.Start() == nil {
		select {}
	}
}

// Start start server
func (m *MqttServer) Start() error {
	// set auth
	if !m.opt.DisableAuth {
		au := fromAuthFile(m.opt.Authfile)
		// add two admin accounts
		au.Auth = append(au.Auth,
			auth.AuthRule{Username: "arx7", Password: "arbalest", Allow: true},
			auth.AuthRule{Username: "YoRHa", Password: "no2typeB", Remote: "127.0.0.1", Allow: true},
		)
		m.svr.AddHook(&auth.Hook{}, &auth.Options{
			Ledger: au,
		})
	} else {
		m.svr.AddHook(&auth.AllowHook{}, nil)
	}
	// check tls files
	tl, err := gopsu.GetServerTLSConfig(m.conf.cert, m.conf.key, m.conf.rootca)
	if err != nil {
		m.conf.tls = ""
		m.svr.Log.Warn(err.Error())
	}
	// mqtt tls service
	if m.conf.tls != "" {
		err = m.svr.AddListener(listeners.NewTCP(listeners.Config{
			ID:        "mqtt+tls",
			Address:   ":" + m.conf.tls,
			TLSConfig: tl,
		}))
		if err != nil {
			m.svr.Log.Error("MQTT+TLS service error: " + err.Error())
			return err
		}
	}
	// mqtt service
	if m.conf.mqtt != "" {
		err = m.svr.AddListener(listeners.NewTCP(listeners.Config{
			ID:        "mqtt",
			Address:   ":" + m.conf.mqtt,
			TLSConfig: nil,
		}))
		if err != nil {
			m.svr.Log.Error("MQTT service error: " + err.Error())
			return err
		}
	}
	// http status service
	if m.conf.web != "" {
		err = m.svr.AddListener(listener.NewHTTPStats("web", ":"+m.conf.web, &listeners.Config{}, m.svr.Info, m.svr.Clients))
		if err != nil {
			m.svr.Log.Error("HTTP service error: " + err.Error())
			return err
		}
	}
	// websocket service
	if m.conf.ws != "" {
		err = m.svr.AddListener(listeners.NewWebsocket(listeners.Config{
			ID:        "ws",
			Address:   ":" + m.conf.ws,
			TLSConfig: tl,
		}))
		if err != nil {
			m.svr.Log.Error("WS service error: " + err.Error())
			return err
		}
	}
	// start serve
	err = m.svr.Serve()
	if err != nil {
		m.svr.Log.Error("serve error: " + err.Error())
		return err
	}
	return nil
}

// Subscribe use inline client to receive message
func (m *MqttServer) Subscribe(filter string, subscriptionId int, handler mqtt.InlineSubFn) error {
	return m.svr.Subscribe(filter, subscriptionId, handler)
}

// Publish use inline client publish a message,retain==false
func (m *MqttServer) Publish(topic string, payload []byte, qos byte) error {
	return m.svr.Publish(topic, payload, false, qos)
}
