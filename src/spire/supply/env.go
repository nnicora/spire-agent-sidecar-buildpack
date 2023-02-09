package supply

import (
	"encoding/json"
	"github.com/nnicora/spire-agent-sidecar-buildpack/src/utils"
	"os"
	"strconv"
	"strings"
)

const (
	vcapEnv                       = "VCAP_SERVICES"
	ztisServiceEnv                = "ZTIS_SERVICE_NAME"
	spireServerAddressEnv         = "SPIRE_SERVER_ADDRESS"
	spireServerPortEnv            = "SPIRE_SERVER_PORT"
	spireEnvoyProxyEnv            = "SPIRE_ENVOY_PROXY"
	spireApplicationSpiffeIdEnv   = "SPIRE_APPLICATION_SPIFFE_ID"
	spireCloudFoundrySVIDStoreEnv = "SPIRE_CLOUDFOUNDRY_SVID_STORE"
)

type ZTIS struct {
	BindingGuid  string       `json:"binding_guid"`
	BindingName  string       `json:"binding_name"`
	InstanceGuid string       `json:"instance_guid"`
	InstanceName string       `json:"instance_name"`
	Label        string       `json:"label"`
	Name         string       `json:"name"`
	Plan         string       `json:"plan"`
	Credentials  *Credentials `json:"credentials"`
}

type Credentials struct {
	Spire    *Spire    `json:"spire"`
	Workload *Workload `json:"workload"`
}

type Spire struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func (s *Credentials) SpireTrustDomain() string {
	spiffeID := s.Workload.SpiffeID
	if strings.HasPrefix(spiffeID, "spiffe://") {
		spiffeID = strings.TrimPrefix(spiffeID, "spiffe://")
		return strings.Split(spiffeID, "/")[0]
	}
	return ""
}

type Workload struct {
	SpiffeID string `json:"spiffeID"`
}

func envWithDefault(key, def string) string {
	value := strings.Trim(os.Getenv(key), " ")
	if value == "" {
		return def
	}
	return value
}

func env(key string) string {
	return strings.Trim(os.Getenv(key), " ")
}

func LoadZTIS() (*ZTIS, error) {
	vcap := env(vcapEnv)

	data := map[string]interface{}{}

	err := json.Unmarshal([]byte(vcap), &data)
	if err != nil {
		return nil, err
	}

	if v, ok := data[envWithDefault(ztisServiceEnv, "ztis")]; ok {
		sdata, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}

		t := struct {
			ZTIS []*ZTIS
		}{}

		err = json.Unmarshal(sdata, &t.ZTIS)
		if err != nil {
			return nil, err
		}

		if len(t.ZTIS) > 0 {
			return t.ZTIS[0], nil
		}

	}

	ssa, err := utils.Env(spireServerAddressEnv)
	if err != nil {
		return nil, err
	}
	spiffe, err := utils.Env(spireApplicationSpiffeIdEnv)
	if err != nil {
		return nil, err
	}

	portStr := envWithDefault(spireServerPortEnv, "8081")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		port = 8081
	}

	ztis := &ZTIS{
		Credentials: &Credentials{
			Spire: &Spire{
				Host: ssa,
				Port: port,
			},
			Workload: &Workload{
				SpiffeID: spiffe,
			},
		},
	}

	return ztis, nil
}
