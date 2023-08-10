package main

import (
	"encoding/json"
	"os"
	"strings"
)

const (
	vcapEnv = "VCAP_SERVICES"
)

type Instance struct {
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

func loadVCAP() (map[string][]*Instance, error) {
	data := map[string][]*Instance{}

	vcap := strings.TrimSpace(os.Getenv(vcapEnv))

	if vcap != "" {
		err := json.Unmarshal([]byte(vcap), &data)
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}

func ExtractSpireCredentials() *Credentials {
	if d, err := loadVCAP(); err != nil {
		logger.Printf("Error: %v", err)
	} else {
		for _, v := range d {
			if len(v) > 0 {
				for _, i := range v {
					if i.Credentials != nil && i.Credentials.Spire != nil && i.Credentials.Workload != nil {
						return i.Credentials
					}
				}
			}
		}
	}

	return nil
}
