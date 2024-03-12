package listener

import (
	"context"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin/render"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/system"
	"github.com/xyzj/gopsu/json"
)

var (
	t1 = `<html lang="zh-cn">
	    <head>
	    <meta content="text/html; charset=utf-8" http-equiv="content-type" />
		<script language="JavaScript">
	      function myrefresh()
	        {
	          window.location.reload();
	        }
	      setTimeout('myrefresh()',180000); //指定180s刷新一次
	    </script>
		{{template "css"}}
		</head>
		{{template "body" .}}
	</html>`
	t2 = `{{define "css"}}
	<style type="text/css">
		a {
	      color: #4183C4;
	      font-size: 16px; }
		h1, h2, h3, h4, h5, h6 {
          margin: 20px 0 10px;
          padding: 0;
          font-weight: bold;
          -webkit-font-smoothing: antialiased;
          cursor: text;
          position: relative; }
		h1 {
	      font-size: 28px;
	      color: black; }
	    h2 {
	      font-size: 24px;
	      border-bottom: 1px solid #cccccc;
	      color: black; }
	    h3 {
	      font-size: 18px; }
	    h4 {
	      font-size: 16px; }
	    h5 {
	      font-size: 14px; }
	    h6 {
	      color: #777777;
	      font-size: 14px; }
	    table {
	      padding: 0; }
  	      table tr {
  	        border-top: 1px solid #cccccc;
  	        background-color: white;
  	        margin: 0;
  	        padding: 0; }
  	        table tr:nth-child(2n) {
  	          background-color: #f8f8f8; }
  	        table tr th {
  	          font-weight: bold;
  	          border: 1px solid #cccccc;
  	          text-align: center;
  	          margin: 0;
  	          padding: 6px 13px; }
  	        table tr td {
  	          border: 1px solid #cccccc;
  	          text-align: center;
  	          margin: 0;
  	          padding: 6px 13px; }
  	        table tr th :first-child, table tr td :first-child {
  	          margin-top: 0; }
  	        table tr th :last-child, table tr td :last-child {
  	          margin-bottom: 0; }
	</style>
	{{end}}`
	t3 = `{{define "body"}}
	<body>
		<h3>服务器时间：</h3><a>{{.timer}}</a>
		<h3>在线设备信息：</h3>
		<table>
			<thead>
			<tr>
			<th>客户端ID</th>
			<th>客户端IP</th>
			<th>协议版本</th>
			<th>协议类型</th>
			<th>订阅信息</th>
			</tr>
			</thead>
			<tbody>
				{{range $idx, $elem := .clients}}
				<tr>
					{{range $key,$value:=$elem}}
						<td>{{$value}}</td>
					{{end}}
				</tr>
				{{end}}
			</tbody>
		</table>
	</body>
	{{end}}`
)

// HTTPStats is a listener for presenting the server $SYS stats on a JSON http endpoint.
type HTTPStats struct {
	sync.RWMutex
	config      *listeners.Config // configuration values for the listener
	listen      *http.Server      // the http server
	sysInfo     *system.Info      // pointers to the server data
	clientsInfo *mqtt.Clients     // pointers to the server data
	log         *slog.Logger      // server logger
	id          string            // the internal id of the listener
	address     string            // the network address to bind to
	end         uint32            // ensure the close methods are only called once
}

// NewHTTPStats initialises and returns a new HTTP listener, listening on an address.
func NewHTTPStats(id, address string, config *listeners.Config, sysInfo *system.Info, cliInfo *mqtt.Clients) *HTTPStats {
	if config == nil {
		config = new(listeners.Config)
	}
	return &HTTPStats{
		id:          id,
		address:     address,
		sysInfo:     sysInfo,
		clientsInfo: cliInfo,
		config:      config,
	}
}

// ID returns the id of the listener.
func (l *HTTPStats) ID() string {
	return l.id
}

// Address returns the address of the listener.
func (l *HTTPStats) Address() string {
	return l.address
}

// Protocol returns the address of the listener.
func (l *HTTPStats) Protocol() string {
	if l.listen != nil && l.listen.TLSConfig != nil {
		return "https"
	}

	return "http"
}

// Init initializes the listener.
func (l *HTTPStats) Init(log *slog.Logger) error {
	l.log = log
	mux := http.NewServeMux()
	mux.HandleFunc("/info", l.infoHandler)
	mux.HandleFunc("/clients", l.clientHandler)
	mux.HandleFunc("/raw", l.debugHandler)
	l.listen = &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		Addr:         l.address,
		Handler:      mux,
	}

	if l.config.TLSConfig != nil {
		l.listen.TLSConfig = l.config.TLSConfig
	}

	return nil
}

// Serve starts listening for new connections and serving responses.
func (l *HTTPStats) Serve(establish listeners.EstablishFn) {

	var err error
	if l.listen.TLSConfig != nil {
		err = l.listen.ListenAndServeTLS("", "")
	} else {
		err = l.listen.ListenAndServe()
	}

	// After the listener has been shutdown, no need to print the http.ErrServerClosed error.
	if err != nil && atomic.LoadUint32(&l.end) == 0 {
		l.log.Error("failed to serve.", "error", err, "listener", l.id)
	}
}

// Close closes the listener and any client connections.
func (l *HTTPStats) Close(closeClients listeners.CloseFn) {
	l.Lock()
	defer l.Unlock()

	if atomic.CompareAndSwapUint32(&l.end, 0, 1) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = l.listen.Shutdown(ctx)
	}

	closeClients(l.id)
}

// clientHandler is an HTTP handler which outputs the $SYS stats as JSON.
func (l *HTTPStats) clientHandler(w http.ResponseWriter, req *http.Request) {
	info := l.clientsInfo.GetAll()
	sss := make([][]string, 0, len(info))
	for _, v := range info {
		var s = make([]string, 0)
		for k, x := range v.State.Subscriptions.GetAll() {
			s = append(s, k+":"+strconv.Itoa(int(x.Qos)))
		}
		sss = append(sss, []string{v.ID, v.Net.Remote, strconv.Itoa(int(v.Properties.ProtocolVersion)), v.Net.Listener, strings.Join(s, ";")})
	}
	sort.Slice(sss, func(i, j int) bool {
		return sss[i][0] < sss[j][0]
	})
	d := map[string]any{
		"timer":   time.Now().String(),
		"clients": sss,
	}
	t, _ := template.New("systemStatus").Parse(t1 + t2 + t3)
	h := render.HTML{
		Name:     "systemStatus",
		Data:     d,
		Template: t,
	}
	h.WriteContentType(w)
	h.Render(w)
}

// infoHandler is an HTTP handler which outputs the $SYS stats as JSON.
func (l *HTTPStats) infoHandler(w http.ResponseWriter, req *http.Request) {
	info := *l.sysInfo.Clone()

	out, err := json.MarshalIndent(info, "", "\t")
	if err != nil {
		_, _ = io.WriteString(w, err.Error())
	}

	_, _ = w.Write(out)
}

// debugHandler is an HTTP handler which outputs the $SYS stats as JSON.
func (l *HTTPStats) debugHandler(w http.ResponseWriter, req *http.Request) {
	info := l.clientsInfo.GetAll()
	for _, v := range info {
		s, err := json.MarshalIndent(v, "", "  ")
		if err == nil {
			w.Write(s)
			w.Write([]byte{10})
		}
	}
}
