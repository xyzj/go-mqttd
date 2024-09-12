package server

import (
	"fmt"
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
	AuthSample = auth.Users{
		"thisisanACLample": {
			Password: "lostjudgment",
			Disallow: true,
			ACL: auth.Filters{
				"deny/#":  auth.Deny,
				"read/#":  auth.ReadOnly,
				"write/#": auth.WriteOnly,
				"rw/#":    auth.ReadWrite,
			},
		},
		"user01": {
			Password: "fallguys",
			ACL: auth.Filters{
				"down/#": auth.ReadOnly,
				"up/#":   auth.WriteOnly,
			},
		},
	}
)

func FromAuthfile(authfile string) (*auth.Ledger, error) {
	ac := auth.Users{}
	if authfile == "" {
		return nil, fmt.Errorf("filename is empty")
	}
	b, err := os.ReadFile(authfile)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(b, &ac)
	if err != nil {
		return nil, err
	}
	return &auth.Ledger{Users: ac}, nil
}

func InitAuthfile(filename string) error {
	b, err := yaml.Marshal(AuthSample)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, b, 0o664)
}
