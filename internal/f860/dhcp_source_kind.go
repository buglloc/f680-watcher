package f860

import (
	"encoding/json"
	"fmt"
	"strconv"

	"gopkg.in/yaml.v3"
)

var _ yaml.Unmarshaler = (*DHCPSourceKind)(nil)
var _ yaml.Marshaler = (*DHCPSourceKind)(nil)

type DHCPSourceKind uint8

const (
	DHCPSourceKindNone     DHCPSourceKind = 0
	DHCPSourceKindLocal    DHCPSourceKind = 1
	DHCPSourceKindInternet DHCPSourceKind = 2
)

func (k DHCPSourceKind) String() string {
	switch k {
	case DHCPSourceKindNone:
		return ""
	case DHCPSourceKindLocal:
		return "local"
	case DHCPSourceKindInternet:
		return "internet"
	default:
		return fmt.Sprintf("unknown_%d", k)
	}
}

func (k *DHCPSourceKind) fromString(s string) error {
	switch s {
	case "":
		*k = DHCPSourceKindNone
	case "local", "1":
		*k = DHCPSourceKindLocal
	case "internet", "2":
		*k = DHCPSourceKindInternet
	default:
		return fmt.Errorf("unknown DHCP source kind: %s", s)
	}
	return nil
}

func (k DHCPSourceKind) Router() string {
	return strconv.Itoa(int(k))
}

func (k *DHCPSourceKind) FromRouter(in string) error {
	return k.fromString(in)
}

func (k DHCPSourceKind) MarshalYAML() (interface{}, error) {
	return k.String(), nil
}

func (k *DHCPSourceKind) UnmarshalYAML(val *yaml.Node) error {
	var s string
	if err := val.Decode(&s); err != nil {
		return err
	}

	return k.fromString(s)
}

func (k DHCPSourceKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}

func (k *DHCPSourceKind) UnmarshalJSON(in []byte) error {
	var s string
	if err := json.Unmarshal(in, &s); err != nil {
		return err
	}

	return k.fromString(s)
}

func (k *DHCPSourceKind) UnmarshalText(in []byte) error {
	return k.fromString(string(in))
}
