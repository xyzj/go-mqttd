package server

import (
	"fmt"
	"go-mqttd/listener"
	"os"
	"strconv"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/xyzj/gopsu"
)

// Opt server option
type Opt struct {
	// tls cert file path
	Cert string
	// tls key file path
	Key string
	// tls root ca file path
	RootCA string
	// Authfile auth config file path
	Authfile string
	// mqtt port
	PortMqtt int
	// mqtt+tls port
	PortTLS int
	// http status port
	PortWeb int
	// websocket port
	PortWS int
	// DisableAuth clients do not need username and password
	DisableAuth bool
	// InsideJob enable or disable inline client
	InsideJob bool
}

// MqttServer a new mqtt server
type MqttServer struct {
	svr *mqtt.Server
	opt *Opt
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
	return &MqttServer{
		svr: svr,
		opt: opt,
	}
}

// Stop close server
func (m *MqttServer) Stop() {
	if m == nil || m.svr == nil {
		return
	}
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
	if m == nil || m.svr == nil {
		return fmt.Errorf("use NewServer() to create a new mqtt server")
	}
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
	tl, err := gopsu.GetServerTLSConfig(m.opt.Cert, m.opt.Key, m.opt.RootCA)
	if err != nil {
		m.opt.PortTLS = 0
		m.svr.Log.Warn(err.Error())
	}
	// mqtt tls service
	if m.opt.PortTLS > 0 && m.opt.PortTLS < 65535 {
		err = m.svr.AddListener(listeners.NewTCP(listeners.Config{
			ID:        "mqtt+tls",
			Address:   ":" + strconv.Itoa(m.opt.PortTLS),
			TLSConfig: tl,
		}))
		if err != nil {
			m.svr.Log.Error("MQTT+TLS service error: " + err.Error())
			return err
		}
	}
	// mqtt service
	if m.opt.PortMqtt > 0 && m.opt.PortMqtt < 65535 {
		err = m.svr.AddListener(listeners.NewTCP(listeners.Config{
			ID:        "mqtt",
			Address:   ":" + strconv.Itoa(m.opt.PortMqtt),
			TLSConfig: nil,
		}))
		if err != nil {
			m.svr.Log.Error("MQTT service error: " + err.Error())
			return err
		}
	}
	// http status service
	if m.opt.PortWeb > 0 && m.opt.PortWeb < 65535 {
		err = m.svr.AddListener(listener.NewHTTPStats("web", ":"+strconv.Itoa(m.opt.PortWeb), &listeners.Config{}, m.svr.Info, m.svr.Clients))
		if err != nil {
			m.svr.Log.Error("HTTP service error: " + err.Error())
			return err
		}
	}
	// websocket service
	if m.opt.PortWS > 0 && m.opt.PortWS < 65535 {
		err = m.svr.AddListener(listeners.NewWebsocket(listeners.Config{
			ID:        "ws",
			Address:   ":" + strconv.Itoa(m.opt.PortWS),
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
