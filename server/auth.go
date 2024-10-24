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
	AuthSample = []byte(`thisisanACLsample:
    password: lostjudgment
    acl:
        deny/#: 0
        read/#: 1
        write/#: 2
        rw/#: 3
    disallow: true
control:
    password: dayone
    acl:
        down/#: 3
        up/#: 3
user01:
    password: fallguys
    acl:
        down/+/user01/#: 1
        up/+/user01/#: 2
        up/#: 0
`)
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
	return &auth.Ledger{Users: ac, Auth: auth.AuthRules{}, ACL: auth.ACLRules{}}, nil
}

func InitAuthfile(filename string) error {
	// b, err := yaml.Marshal(AuthSample)
	// if err != nil {
	// 	return err
	// }
	return os.WriteFile(filename, AuthSample, 0o664)
}
