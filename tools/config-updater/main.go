package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

var (
	spireAgentConfig string
	envoyConfig      string
	syncInterval     int
	logger           = log.New(os.Stdout, "debug", 1)
)

func init() {

	flag.StringVar(&spireAgentConfig, "spire-agent-config", "", "path to spire agent config file")
	flag.StringVar(&envoyConfig, "envoy-config", "", "path to envoy config file")
	flag.IntVar(&syncInterval, "sync-interval", 1, "Sync interval; 1 sec as default")
	flag.Parse()
}

func main() {
	for {
		if c := ExtractSpireCredentials(); c != nil {

			if spireAgentConfig != "" {
				err := updateConfigFile(spireAgentConfig, func(s string) string {
					s = strings.ReplaceAll(s, "server_address = \"\"", fmt.Sprintf("server_address = \"%s\"", c.Spire.Host))
					s = strings.ReplaceAll(s, "server_port = 0", fmt.Sprintf("server_port = %d", c.Spire.Port))
					return strings.ReplaceAll(s, "trust_domain = \"\"", fmt.Sprintf("trust_domain = \"%s\"", c.SpireTrustDomain()))
				})
				if err != nil {
					logger.Printf("%v", err)
				}
			}
			if envoyConfig != "" {
				err := updateConfigFile(envoyConfig, func(s string) string {
					return strings.ReplaceAll(s, "- name: \"SpiffeID\"", fmt.Sprintf("- name: \"%s\"", c.Workload.SpiffeID))
				})
				if err != nil {
					logger.Printf("%v", err)
				}
			}
		}

		select {
		case <-time.After(time.Duration(syncInterval) * time.Second):
			//nothing
		}
	}
}

func updateConfigFile(configFilePath string, f func(string) string) error {
	_, err := os.Stat(configFilePath)
	if os.IsNotExist(err) {
		return nil
	}

	if data, err := os.ReadFile(configFilePath); err != nil {
		return fmt.Errorf("failed reading data from file: %v", err)
	} else {
		file, err := os.OpenFile(configFilePath, os.O_RDWR, 0644)
		if err != nil {
			return fmt.Errorf("failed opening file: %v", err)
		} else {
			strData := f(string(data))
			_, err = file.WriteAt([]byte(strData), 0) // Write at 0 beginning
			if err != nil {
				return fmt.Errorf("failed writing to file: %v", err)
			}
			defer file.Close()
			return nil
		}
	}
}
