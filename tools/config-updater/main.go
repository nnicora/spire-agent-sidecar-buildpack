package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
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

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	go updater()

	<-sigint
}

func updater() {
	var reServerAddress = regexp.MustCompile(`server_address(?:.*)=(.*)`)
	var reServerPort = regexp.MustCompile(`server_port(?:.*)=(.*)`)
	var reTrustDomain = regexp.MustCompile(`trust_domain(?:.*)=(.*)`)

	interval := time.Duration(syncInterval) * time.Second

	ticker := time.NewTicker(interval)
	for {
		ticker.Reset(interval)

		c, err := ExtractSpireCredentials()
		if err != nil || c == nil {
			select {
			case <-ticker.C:
			}

			continue
		}

		if spireAgentConfig != "" {
			err := updateConfigFile(spireAgentConfig, func(s string) string {

				matches := reServerAddress.FindStringSubmatch(s)
				if len(matches) > 0 {
					s = strings.ReplaceAll(s, matches[0], fmt.Sprintf("server_address = \"%s\"", c.Spire.Host))
				}

				matches = reServerPort.FindStringSubmatch(s)
				if len(matches) > 0 {
					s = strings.ReplaceAll(s, matches[0], fmt.Sprintf("server_port = %d", c.Spire.Port))
				}

				matches = reTrustDomain.FindStringSubmatch(s)
				if len(matches) > 0 {
					s = strings.ReplaceAll(s, matches[0], fmt.Sprintf("trust_domain = \"%s\"", c.SpireTrustDomain()))
				}

				return s
			})
			if err != nil {
				logger.Printf("couldn't update the spire agent configuration%v", err)
			}
		}

		select {
		case <-ticker.C:
		}
	}
}

func updateConfigFile(configFilePath string, f func(string) string) error {
	_, err := os.Stat(configFilePath)
	if os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("failed reading data from file: %v", err)
	}

	file, err := os.OpenFile(configFilePath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed opening file: %v", err)
	}
	defer func() {
		_ = file.Close()
	}()

	strData := f(string(data))
	_, err = file.WriteAt([]byte(strData), 0) // Write at 0 beginning
	if err != nil {
		return fmt.Errorf("failed writing to file: %v", err)
	}

	return nil
}
