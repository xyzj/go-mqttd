package server

import (
	"os"

	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"gopkg.in/yaml.v3"
)

/*
	Filters Access :

0-Deny      				// user cannot access the topic
1-ReadOnly                // user can only subscribe to the topic
2-WriteOnly               // user can only publish to the topic
3-ReadWrite               // user can both publish and subscribe to the topic
*/
var (
	authSample = &auth.Ledger{
		Users: map[string]auth.UserRule{
			"mqttdevices": {
				Username: "mqttdevices",
				Password: "fallguys",
				ACL: auth.Filters{
					"down/#": auth.ReadOnly,
					"up/#":   auth.WriteOnly,
				},
			},
			"lostjudgment": {
				Username: "lostjudgment",
				Password: "yagami",
				ACL: auth.Filters{
					"deny/#":  auth.Deny,
					"read/#":  auth.ReadOnly,
					"write/#": auth.WriteOnly,
					"rw/#":    auth.ReadWrite,
				},
			},
		},
	}
)

var userMap = map[string]string{
	"arx7": "arbalest",
}

func fromAuthFile(authfile string) *auth.Ledger {
	ac := &auth.Ledger{}
	if authfile == "" {
		return ac
	}
	b, err := os.ReadFile(authfile)
	if err != nil {
		createAuthFile(authfile)
		return ac
	}

	err = yaml.Unmarshal(b, &ac)
	if err != nil {
		createAuthFile(authfile)
		return authSample
	}
	return ac
}

func createAuthFile(filename string) {
	b, err := yaml.Marshal(authSample)
	if err != nil {
		println(err.Error())
		return
	}
	err = os.WriteFile(filename, b, 0o664)
	if err != nil {
		println(err.Error())
	}
}
