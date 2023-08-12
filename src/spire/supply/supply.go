package supply

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"

	"github.tools.sap/pse/spire-agent-sidecar-buildpack/src/utils"
)

const (
	spireServerAddressEnv         = "SPIRE_SERVER_ADDRESS"
	spireServerPortEnv            = "SPIRE_SERVER_PORT"
	spireTrustDomainEnv           = "SPIRE_TRUST_DOMAIN"
	spireBundlePathEnv            = "SPIRE_BUNDLE_PATH"
	spireLogLevelEnv              = "SPIRE_LOG_LEVEL"
	spireCloudFoundrySVIDStoreEnv = "SPIRE_CLOUDFOUNDRY_SVID_STORE"
	svidKeyTypeEnv                = "SPIRE_AGENT_WORKLOAD_X509_SVID_KEY_TYPE"
)

const (
	defaultBundlePathPattern = "/home/vcap/deps/%s/certificates/bundle.crt"
)

var (
	defaultSvidKeyType  = "ec-p256"
	allowedSvidKeyTypes = map[string]struct{}{
		"rsa-2048": {},
		"ec-p256":  {},
	}
)

type Command interface {
	Execute(string, io.Writer, io.Writer, string, ...string) error
	Output(string, string, ...string) (string, error)
	Run(cmd *exec.Cmd) error
}

type Manifest interface {
	DefaultVersion(depName string) (libbuildpack.Dependency, error)
	AllDependencyVersions(string) []string
	RootDir() string
}

type Installer interface {
	InstallDependency(dep libbuildpack.Dependency, outputDir string) error
	InstallOnlyVersion(string, string) error
}

type Stager interface {
	AddBinDependencyLink(string, string) error
	BuildDir() string
	DepDir() string
	DepsDir() string
	DepsIdx() string
	WriteConfigYml(interface{}) error
	WriteEnvFile(string, string) error
	WriteProfileD(string, string) error
}

type Config struct {
	SpireAgent SpireAgentConfig `yaml:"spire-agent"`
	Dist       string           `yaml:"dist"`
}

type SpireAgentConfig struct {
	Version string `yaml:"version"`
}

type Supplier struct {
	Stager       Stager
	Manifest     Manifest
	Installer    Installer
	Log          *libbuildpack.Logger
	Config       Config
	Command      Command
	VersionLines map[string]string
}

func New(stager Stager, manifest Manifest, installer Installer, logger *libbuildpack.Logger, command Command) *Supplier {
	return &Supplier{
		Stager:    stager,
		Manifest:  manifest,
		Installer: installer,
		Log:       logger,
		Command:   command,
	}
}

func (s *Supplier) Run() error {
	s.Log.BeginStep("Supplying SPIRE agent")

	creds, err := s.ExtractSpireCredentialsFromVcapServices()
	if err != nil {
		s.Log.Info("Couldn't load %s environment variable: %v", vcapEnv, err)
	}

	if err := s.Copy("certificates", "certificates"); err != nil {
		s.Log.Error("Failed to copy certificates; %s", err.Error())
		return err
	}

	if err := s.CreateSpireAgentConf(creds); err != nil {
		s.Log.Error("Failed to configure spire-agent.conf file; %s", err.Error())
		return err
	}

	if err := s.Copy("bin", "binaries"); err != nil {
		s.Log.Error("Failed to copy binaries; %s", err.Error())
		return err
	}

	if err := s.CreateLaunchForSidecars(creds); err != nil {
		s.Log.Error("Failed to create the sidecar processes; %s", err.Error())
		return err
	}

	if err := s.Setup(); err != nil {
		s.Log.Error("Could not setup; %s", err.Error())
		return err
	}

	return nil
}

func (s *Supplier) Copy(dst string, srcs ...string) error {
	paths := make([]string, 0, len(srcs)+1)
	paths = append(paths, s.Manifest.RootDir())
	paths = append(paths, srcs...)

	dir := filepath.Join(paths...)

	err := filepath.Walk(dir, func(srcPath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if err != nil {
			s.Log.Error("Can't copy file: %s", err.Error())
			return err
		}
		dstPath := filepath.Join(s.Stager.DepDir(), dst, info.Name())
		if errCopy := libbuildpack.CopyFile(srcPath, dstPath); errCopy != nil {
			s.Log.Error("Can't copy file: %s; Source `%s`, destination `%s`", errCopy.Error(), srcPath, dstPath)
			return errCopy
		}

		s.Log.Info("Copy file from Source `%s`, destination `%s`", srcPath, dstPath)

		return nil
	})
	return err
}

func (s *Supplier) CreateLaunchForSidecars(creds *Credentials) error {
	launch := filepath.Join(s.Stager.DepDir(), "launch.yml")
	if _, err := libbuildpack.FileExists(launch); err != nil {
		return err
	}

	launchFile, err := os.Create(launch)
	if err != nil {
		return err
	}

	_, err = launchFile.WriteString("---\nprocesses:\n")
	if err != nil {
		return err
	}

	spireAgentSidecarTmpl := filepath.Join(s.Manifest.RootDir(), "templates", "spire_agent-sidecar.tmpl")
	spireAgentSidecar := template.Must(template.ParseFiles(spireAgentSidecarTmpl))
	err = spireAgentSidecar.Execute(launchFile, map[string]interface{}{
		"Idx": s.Stager.DepsIdx(),
	})
	if err != nil {
		return err
	}

	svidFile := utils.EnvWithDefault(spireCloudFoundrySVIDStoreEnv, "false")
	if strings.ToLower(svidFile) == "true" {
		svidFileSidecarTmpl := filepath.Join(s.Manifest.RootDir(), "templates", "svid-file-sidecar.tmpl")
		svidFileSidecar := template.Must(template.ParseFiles(svidFileSidecarTmpl))
		err = svidFileSidecar.Execute(launchFile, map[string]interface{}{
			"Idx": s.Stager.DepsIdx(),
		})
		if err != nil {
			return err
		}
	}

	if creds == nil {
		configUpdatersSidecarTmpl := filepath.Join(s.Manifest.RootDir(), "templates", "config-updaters.tmpl")
		configUpdaterSidecar := template.Must(template.ParseFiles(configUpdatersSidecarTmpl))
		err = configUpdaterSidecar.Execute(launchFile, map[string]interface{}{
			"Idx": s.Stager.DepsIdx(),
		})
		if err != nil {
			return err
		}
	}

	err = launchFile.Close()
	return err
}

func (s *Supplier) CreateSpireAgentConf(creds *Credentials) error {
	conf := filepath.Join(s.Stager.DepDir(), "spire-agent.conf")
	if _, err := libbuildpack.FileExists(conf); err != nil {
		return err
	}

	f, err := os.Create(conf)
	if err != nil {
		return err
	}

	s.Log.Info("SPIRE agent configuration: %s", conf)

	confTmpl := filepath.Join(s.Manifest.RootDir(), "templates", "spire-agent-conf.tmpl")
	t := template.Must(template.ParseFiles(confTmpl))

	ssa := utils.EnvWithDefault(spireServerAddressEnv, "")
	ssp := utils.EnvWithDefault(spireServerPortEnv, "0")
	std := utils.EnvWithDefault(spireTrustDomainEnv, "")
	skt := utils.EnvWithDefault(svidKeyTypeEnv, defaultSvidKeyType)
	if _, ok := allowedSvidKeyTypes[skt]; !ok {
		skt = defaultSvidKeyType
	}

	if creds != nil && creds.Spire != nil {
		ssa = creds.Spire.Host
		ssp = fmt.Sprintf("%d", creds.Spire.Port)
		std = creds.SpireTrustDomain()
	}

	ll := utils.EnvWithDefault(spireLogLevelEnv, "INFO")

	bundlePath := utils.EnvWithDefault(spireBundlePathEnv, fmt.Sprintf(defaultBundlePathPattern, s.Stager.DepsIdx()))
	if _, err := os.Stat(bundlePath); err != nil {
		if os.IsNotExist(err) {
			return err
		}
	}
	data := map[string]interface{}{
		"BundlePath":         bundlePath,
		"SpireServerAddress": ssa,
		"SpireServerPort":    ssp,
		"TrustDomain":        std,
		"SvidKeyType":        skt,
		"LogLevel":           ll,
	}

	err = t.Execute(f, data)
	if err != nil {
		return err
	}

	err = f.Close()
	return err
}

func (s *Supplier) Setup() error {
	var m struct {
		VersionLines map[string]string `yaml:"version_lines"`
	}
	if err := libbuildpack.NewYAML().Load(filepath.Join(s.Manifest.RootDir(), "manifest.yml"), &m); err != nil {
		return err
	}
	s.VersionLines = m.VersionLines

	// create logs directory in case if doesn't exist
	logsDirPath := filepath.Join(s.Stager.BuildDir(), "logs")
	exists, err := libbuildpack.FileExists(logsDirPath)
	if err != nil {
		return err
	}

	if !exists {
		if err := os.MkdirAll(logsDirPath, os.ModePerm); err != nil {
			s.Log.Error("could not create 'logs' directory: %v", err.Error())
		}
	}

	return nil
}
