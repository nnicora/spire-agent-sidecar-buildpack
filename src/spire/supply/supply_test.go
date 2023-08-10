package supply_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	"github.com/golang/mock/gomock"

	"github.tools.sap/pse/spire-agent-sidecar-buildpack/src/spire/supply"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	credentialVCAP = `
		{
			"zero-trust-identity": [
				{
					"label": "zero-trust-identity",
					"provider": null,
					"plan": "standard",
					"name": "testing",
					"tags": [],
					"instance_guid": "c30ce7a4-3e4b-4846-97c7-3b58dbfaef01",
					"instance_name": "testing",
					"binding_guid": "e0601b74-7852-4923-80ed-dde88cdf9b4d",
					"binding_name": null,
					"credentials": {
						"spire": {
							"host": "spire-t54c68756t.0trust.net.sap",
							"port": 8081
						},
						"workload": {
							"spiffeID": "spiffe://0trust.net.sap/v01/a9bd8697cfbba44eacef1be9e9e2c10afb1562a980297b6edaf65efb9aedb746"
						},
						"parameters": {
							"app-identifier": ""
						}
					},
					"syslog_drain_url": null,
					"volume_mounts": []
				}
			]
		}`

	noCredentialsVCAP = `
		{
			"zero-trust-identity": [
				{
					"label": "zero-trust-identity",
					"provider": null,
					"plan": "standard",
					"name": "testing",
					"tags": [],
					"instance_guid": "c30ce7a4-3e4b-4846-97c7-3b58dbfaef01",
					"instance_name": "testing",
					"binding_guid": "e0601b74-7852-4923-80ed-dde88cdf9b4d",
					"binding_name": null,
					"syslog_drain_url": null,
					"volume_mounts": []
				}
			]
		}`

	spreAgentConf = `agent {
  server_address = "spire-t54c68756t.0trust.net.sap"
  server_port = 8081
  log_level = "INFO"
  trust_domain = "0trust.net.sap"
  trust_bundle_path = "/home/vcap/deps/0/certificates/bundle.crt"

  workload_x509_svid_key_type = "ec-p256"
}

plugins {
  KeyManager "memory" {
    plugin_data {}
  }

  NodeAttestor "cf_iic" {
    plugin_cmd = "/home/vcap/deps/0/bin/cf_iic"
    plugin_data {
      private_key_path = "/etc/cf-instance-credentials/instance.key"
      certificate_path = "/etc/cf-instance-credentials/instance.crt"
    }
  }
  
  WorkloadAttestor "unix" {}
}
`
	launchSpireAgentConf = `---
processes:
- type: "spire_agent"
  command: "/home/vcap/deps/0/bin/spire-agent run -config /home/vcap/deps/0/spire-agent.conf"
  platforms:
    cloudfoundry:
      sidecar_for: [ "web"]
`
	launchSpireAgentSvidStoreConf = `---
processes:
- type: "spire_agent"
  command: "/home/vcap/deps/0/bin/spire-agent run -config /home/vcap/deps/0/spire-agent.conf"
  platforms:
    cloudfoundry:
      sidecar_for: [ "web"]
- type: "svid-file-script"
  command: "mkdir -p /tmp/spire-agent/certificates && while (true); do echo 'Refresh SVID'; /home/vcap/deps/0/bin/spire-agent api fetch x509 -socketPath /tmp/spire-agent/public/api.sock -write /tmp/spire-agent/certificates; sleep 10; done"
  platforms:
    cloudfoundry:
      sidecar_for: [ "web"]
`

	launchSpireAgentConfiUpdaterConf = `---
processes:
- type: "spire_agent"
  command: "/home/vcap/deps/0/bin/spire-agent run -config /home/vcap/deps/0/spire-agent.conf"
  platforms:
    cloudfoundry:
      sidecar_for: [ "web"]
- type: "config-updater"
  command: "/home/vcap/deps/0/bin/config-updater -spire-agent-config /home/vcap/deps/0/spire-agent.conf -envoy-config /home/vcap/deps/0/envoy-config.yaml -sync-interval 1"
  platforms:
    cloudfoundry:
      sidecar_for: [ "web"]
`
	launchSpireAgentSvidStoreScriptConfiUpdaterConf = `---
processes:
- type: "spire_agent"
  command: "/home/vcap/deps/0/bin/spire-agent run -config /home/vcap/deps/0/spire-agent.conf"
  platforms:
    cloudfoundry:
      sidecar_for: [ "web"]
- type: "svid-file-script"
  command: "mkdir -p /tmp/spire-agent/certificates && while (true); do echo 'Refresh SVID'; /home/vcap/deps/0/bin/spire-agent api fetch x509 -socketPath /tmp/spire-agent/public/api.sock -write /tmp/spire-agent/certificates; sleep 10; done"
  platforms:
    cloudfoundry:
      sidecar_for: [ "web"]
- type: "config-updater"
  command: "/home/vcap/deps/0/bin/config-updater -spire-agent-config /home/vcap/deps/0/spire-agent.conf -envoy-config /home/vcap/deps/0/envoy-config.yaml -sync-interval 1"
  platforms:
    cloudfoundry:
      sidecar_for: [ "web"]
`
)

//go:generate mockgen -source=supply.go --destination=mocks_test.go --package=supply_test

var _ = Describe("Supply", func() {
	var (
		bpDir         string
		buildDir      string
		depsDir       string
		depsIdx       string
		gs            *supply.Supplier
		logger        *libbuildpack.Logger
		buffer        *bytes.Buffer
		err           error
		mockCtrl      *gomock.Controller
		mockInstaller *MockInstaller
	)

	BeforeEach(func() {
		bpDir = initPackageDir()
		bpDir = filepath.Dir(bpDir)
		bpDir = filepath.Dir(bpDir)
		bpDir = filepath.Dir(bpDir)

		Expect(os.WriteFile(filepath.Join(bpDir, "VERSION"), []byte("0.2.0"), 0644)).To(Succeed())

		buildDir, err = os.MkdirTemp("", "buildpack.build.")
		Expect(err).To(BeNil())

		depsDir, err = os.MkdirTemp("", "buildpack.deps.")
		Expect(err).To(BeNil())

		depsIdx = "0"

		err = os.MkdirAll(filepath.Join(depsDir, depsIdx), 0755)
		Expect(err).To(BeNil())

		buffer = new(bytes.Buffer)

		logger = libbuildpack.NewLogger(ansicleaner.New(buffer))

		mockCtrl = gomock.NewController(GinkgoT())
		mockInstaller = NewMockInstaller(mockCtrl)
	})

	JustBeforeEach(func() {
		args := []string{buildDir, "", depsDir, depsIdx}

		manifest, err := libbuildpack.NewManifest(bpDir, logger, time.Now())
		Expect(err).To(BeNil())
		stager := libbuildpack.NewStager(args, logger, manifest)

		gs = &supply.Supplier{
			Stager:    stager,
			Manifest:  manifest,
			Installer: mockInstaller,
			Log:       logger,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
		Expect(os.RemoveAll(buildDir)).To(Succeed())
		Expect(os.RemoveAll(depsDir)).To(Succeed())
	})

	Context("ExtractSpireCredentialsFromVcapServices", func() {

		AfterEach(func() {
			err := os.Unsetenv("VCAP_SERVICES")
			Expect(err).To(BeNil())
		})

		It("no vcap has been found", func() {
			vcap, err := gs.ExtractSpireCredentialsFromVcapServices()

			Expect(err).To(BeNil())
			Expect(vcap).To(BeNil())
		})

		It("empty vcap", func() {
			err := os.Setenv("VCAP_SERVICES", "{}")
			Expect(err).To(BeNil())

			vcap, err := gs.ExtractSpireCredentialsFromVcapServices()

			Expect(err).To(BeNil())
			Expect(vcap).To(BeNil())
		})

		It("invalid vcap content", func() {
			err := os.Setenv("VCAP_SERVICES", "{ buuu {} }")
			Expect(err).To(BeNil())

			vcap, err := gs.ExtractSpireCredentialsFromVcapServices()

			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(Equal("invalid character 'b' looking for beginning of object key string"))

			Expect(vcap).To(BeNil())
		})

		It("no credentials vcap content", func() {
			err := os.Setenv("VCAP_SERVICES", noCredentialsVCAP)
			Expect(err).To(BeNil())

			vcap, err := gs.ExtractSpireCredentialsFromVcapServices()
			Expect(err).To(BeNil())
			Expect(vcap).To(BeNil())

		})

		It("valid vcap withcredential scontent", func() {
			err := os.Setenv("VCAP_SERVICES", credentialVCAP)
			Expect(err).To(BeNil())

			vcap, err := gs.ExtractSpireCredentialsFromVcapServices()
			Expect(err).To(BeNil())
			Expect(vcap).NotTo(BeNil())

			Expect(vcap.Spire).NotTo(BeNil())
			Expect(vcap.Spire.Host).To(Equal("spire-t54c68756t.0trust.net.sap"))
			Expect(vcap.Spire.Port).To(Equal(8081))
			Expect(vcap.Workload).NotTo(BeNil())
			Expect(vcap.Workload.SpiffeID).To(Equal("spiffe://0trust.net.sap/v01/a9bd8697cfbba44eacef1be9e9e2c10afb1562a980297b6edaf65efb9aedb746"))
		})
	})

	Context("Copy Certificates", func() {
		AfterEach(func() {
			err := os.Unsetenv("VCAP_SERVICES")
			Expect(err).To(BeNil())
		})

		It("success", func() {
			certificatesToBeCopied := map[string]bool{
				"bundle.crt": false,
			}

			err := gs.Copy("certificates", "certificates")
			Expect(err).To(BeNil())

			dir := filepath.Join(gs.Stager.DepDir(), "certificates")
			err = filepath.Walk(dir, func(srcPath string, info os.FileInfo, err error) error {
				if info.IsDir() {
					return nil
				}

				if _, ok := certificatesToBeCopied[info.Name()]; ok {
					certificatesToBeCopied[info.Name()] = true
				}

				return nil
			})
			Expect(err).To(BeNil())

			for _, v := range certificatesToBeCopied {
				Expect(v).To(Equal(true))
			}
		})
	})

	Context("Copy Binaries", func() {
		AfterEach(func() {
			err := os.Unsetenv("VCAP_SERVICES")
			Expect(err).To(BeNil())
		})

		It("success", func() {
			binariesToBeCopied := map[string]bool{
				"config-updater": false,
				"spire-agent":    false,
				"cf_iic":         false,
			}

			err := gs.Copy("bin", "binaries")
			Expect(err).To(BeNil())

			dir := filepath.Join(gs.Stager.DepDir(), "bin")
			err = filepath.Walk(dir, func(srcPath string, info os.FileInfo, err error) error {
				if info.IsDir() {
					return nil
				}

				if _, ok := binariesToBeCopied[info.Name()]; ok {
					binariesToBeCopied[info.Name()] = true
				}

				return nil
			})
			Expect(err).To(BeNil())

			for _, v := range binariesToBeCopied {
				Expect(v).To(Equal(true))
			}
		})
	})

	Context("CreateSpireAgentConf", func() {
		AfterEach(func() {
			err = os.Unsetenv("VCAP_SERVICES")
			Expect(err).To(BeNil())
		})

		It("success", func() {
			err := os.Setenv("VCAP_SERVICES", credentialVCAP)
			Expect(err).To(BeNil())

			creds, err := gs.ExtractSpireCredentialsFromVcapServices()
			Expect(err).To(BeNil())
			Expect(creds).NotTo(BeNil())

			err = gs.CreateSpireAgentConf(creds)
			Expect(err).To(BeNil())

			conf := filepath.Join(gs.Stager.DepDir(), "spire-agent.conf")
			data, err := os.ReadFile(conf)
			Expect(err).To(BeNil())
			Expect(data).NotTo(BeNil())
			Expect(data).To(Equal([]byte(spreAgentConf)))
		})
	})

	Context("CreateLaunchForSidecars", func() {

		AfterEach(func() {
			err := os.Unsetenv("SPIRE_CLOUDFOUNDRY_SVID_STORE")
			Expect(err).To(BeNil())

			err = os.Unsetenv("VCAP_SERVICES")
			Expect(err).To(BeNil())
		})

		It("spire agent launcher", func() {
			err := os.Setenv("VCAP_SERVICES", credentialVCAP)
			Expect(err).To(BeNil())

			creds, err := gs.ExtractSpireCredentialsFromVcapServices()
			Expect(err).To(BeNil())
			Expect(creds).NotTo(BeNil())

			err = gs.CreateLaunchForSidecars(creds)
			Expect(err).To(BeNil())

			launch := filepath.Join(gs.Stager.DepDir(), "launch.yml")
			data, err := os.ReadFile(launch)
			Expect(err).To(BeNil())
			Expect(data).NotTo(BeNil())
			Expect(data).To(Equal([]byte(launchSpireAgentConf)))
		})

		It("spire agent and svid store script launcher", func() {

			err := os.Setenv("SPIRE_CLOUDFOUNDRY_SVID_STORE", "true")
			Expect(err).To(BeNil())

			err = os.Setenv("VCAP_SERVICES", credentialVCAP)
			Expect(err).To(BeNil())

			creds, err := gs.ExtractSpireCredentialsFromVcapServices()
			Expect(err).To(BeNil())
			Expect(creds).NotTo(BeNil())

			err = gs.CreateLaunchForSidecars(creds)
			Expect(err).To(BeNil())

			launch := filepath.Join(gs.Stager.DepDir(), "launch.yml")
			data, err := os.ReadFile(launch)
			Expect(err).To(BeNil())
			Expect(data).NotTo(BeNil())
			Expect(data).To(Equal([]byte(launchSpireAgentSvidStoreConf)))
		})

		It("spire agent with config updater script launcher", func() {
			err = gs.CreateLaunchForSidecars(nil)
			Expect(err).To(BeNil())

			launch := filepath.Join(gs.Stager.DepDir(), "launch.yml")
			data, err := os.ReadFile(launch)
			Expect(err).To(BeNil())
			Expect(data).NotTo(BeNil())
			Expect(data).To(Equal([]byte(launchSpireAgentConfiUpdaterConf)))
		})

		It("spire agent, config updater, svid store script launcher", func() {

			err := os.Setenv("SPIRE_CLOUDFOUNDRY_SVID_STORE", "true")
			Expect(err).To(BeNil())

			err = gs.CreateLaunchForSidecars(nil)
			Expect(err).To(BeNil())

			launch := filepath.Join(gs.Stager.DepDir(), "launch.yml")
			data, err := os.ReadFile(launch)
			Expect(err).To(BeNil())
			Expect(data).NotTo(BeNil())
			Expect(data).To(Equal([]byte(launchSpireAgentSvidStoreScriptConfiUpdaterConf)))
		})
	})

	Context("All", func() {
		It("success run", func() {
			err := os.Setenv("SPIRE_CLOUDFOUNDRY_SVID_STORE", "true")
			Expect(err).To(BeNil())

			err = os.Setenv("VCAP_SERVICES", credentialVCAP)
			Expect(err).To(BeNil())

			err = gs.Run()
			Expect(err).To(BeNil())

			// verify launch content
			launch := filepath.Join(gs.Stager.DepDir(), "launch.yml")
			dataLaunch, err := os.ReadFile(launch)
			Expect(err).To(BeNil())
			Expect(dataLaunch).NotTo(BeNil())
			Expect(dataLaunch).To(Equal([]byte(launchSpireAgentSvidStoreConf)))

			// verify spire agent config
			conf := filepath.Join(gs.Stager.DepDir(), "spire-agent.conf")
			dataSpireConfig, err := os.ReadFile(conf)
			Expect(err).To(BeNil())
			Expect(dataSpireConfig).NotTo(BeNil())
			Expect(dataSpireConfig).To(Equal([]byte(spreAgentConf)))

			// verify binaries if are copied
			binariesToBeCopied := map[string]bool{
				"config-updater": false,
				"spire-agent":    false,
				"cf_iic":         false,
			}

			dirBin := filepath.Join(gs.Stager.DepDir(), "bin")
			err = filepath.Walk(dirBin, func(srcPath string, info os.FileInfo, err error) error {
				if info.IsDir() {
					return nil
				}

				if _, ok := binariesToBeCopied[info.Name()]; ok {
					binariesToBeCopied[info.Name()] = true
				}

				return nil
			})
			Expect(err).To(BeNil())

			for _, v := range binariesToBeCopied {
				Expect(v).To(Equal(true))
			}

			// veiofy if certificates are copied
			certificatesToBeCopied := map[string]bool{
				"bundle.crt": false,
			}

			dirCertificates := filepath.Join(gs.Stager.DepDir(), "certificates")
			err = filepath.Walk(dirCertificates, func(srcPath string, info os.FileInfo, err error) error {
				if info.IsDir() {
					return nil
				}

				if _, ok := certificatesToBeCopied[info.Name()]; ok {
					certificatesToBeCopied[info.Name()] = true
				}

				return nil
			})
			Expect(err).To(BeNil())

			for _, v := range certificatesToBeCopied {
				Expect(v).To(Equal(true))
			}
		})
	})

})

func initPackageDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("unable to obtain caller information")
	}
	return filepath.Dir(file)
}
