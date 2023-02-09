package supply

import (
	"os"
	"testing"
)

func TestZtisSuccess(t *testing.T) {
	vcap := `{
 "ztis": [
  {
   "binding_guid": "54b87746-e348-43e0-ae63-dd494bf3be9c",
   "binding_name": null,
   "credentials": {
    "spire": {
     "host": "spire-t988db889d.ingress.dev-i542315.mesh-dev.shoot.canary.k8s-hana.ondemand.com",
     "port": 8081
    },
    "workload": {
     "spiffeID": "spiffe://staging.0trust.net.sap/v01/385a880b4541179838978f570b0d3beecd3cac4240f53baa803cab44517d96be"
    }
   },
   "instance_guid": "3842a3c2-0ff0-4cae-bbd7-2f06e826b255",
   "instance_name": "ztis-app-identity",
   "label": "ztis",
   "name": "ztis-app-identity",
   "plan": "app-identity",
   "provider": null,
   "syslog_drain_url": null,
   "tags": [],
   "volume_mounts": []
  }
 ]
}
`
	os.Setenv("VCAP_SERVICES", vcap)

	d, err := LoadZTIS()
	if err != nil {
		t.Errorf("can't parse environment variable")
	}

	if d.Credentials == nil {
		t.Errorf("credentials not found")
	}
	if d.Credentials.SpireTrustDomain() != "staging.0trust.net.sap" {
		t.Errorf("incorrect trust domain")
	}
}

func TestZtisNoData(t *testing.T) {
	os.Setenv("VCAP_SERVICES", `{}`)

	d, err := LoadZTIS()
	if err != nil {
		t.Errorf("can't parse environment variable")
	}

	if d.Credentials == nil {
		t.Errorf("credentials not found")
	}
	if d.Credentials.SpireTrustDomain() != "" {
		t.Errorf("incorrect trust domain")
	}
}
