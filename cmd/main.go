package main

import (
	"flag"
	"gomqttd/server"

	"github.com/xyzj/gopsu"
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

func main() {
	svr := server.NewServer(&server.Opt{
		DisableAuth: *disableAuth,
		Confile: func(name string) string {
			if name == "" {
				return pathtool.JoinPathFromHere(confname)
			}
			return name
		}(*confile),
		Authfile: *authfile,
	})
	gocmd.DefaultProgram(&gocmd.Info{
		Ver:      "Core ver: " + cover + "\nGo ver:   " + gover,
		Title:    "golang mqtt broker",
		Descript: "based on mochi-mqtt, support MQTT v3.11 and MQTT v5.0",
	}).AddCommand(&gocmd.Command{
		Name:     "default-config",
		Descript: "create default config file",
		RunWithExitCode: func(pi *gocmd.ProcInfo) int {
			if err := svr.SaveConfig(); err != nil {
				println("save config file error: " + err.Error())
				return 1
			}
			println("file save as " + confname)
			return 0
		},
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
			if err := c.CreateCert([]string{"*.wlst.vip", "localhost"}, ips); err != nil {
				println(err.Error())
				return 1
			}
			println("done.")
			return 0
		},
	}).AfterStop(func() {
		svr.Stop()
	}).Execute()
	svr.Run()
}
