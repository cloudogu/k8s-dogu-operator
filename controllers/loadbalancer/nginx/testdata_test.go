package nginx

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
)

//go:embed testdata/ldap-dogu.json
var ldapBytes []byte

//go:embed testdata/ldap-only-udp-dogu.json
var ldapOnlyUDPBytes []byte

func readLdapDogu(t *testing.T) *core.Dogu {
	t.Helper()

	data := &core.Dogu{}
	err := json.Unmarshal(ldapBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func readLdapDoguOnlyUDP(t *testing.T) *core.Dogu {
	t.Helper()

	data := &core.Dogu{}
	err := json.Unmarshal(ldapOnlyUDPBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}
