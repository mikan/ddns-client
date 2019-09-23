// Copyright 2017-2019 mikan.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

// ClassName defines supported class names.
type ClassName string

const (
	ValueDomain = ClassName("valuedomain")
	Cloudflare  = ClassName("cloudflare")
)

// JSON "target" element
type Target struct {
	Class    ClassName `json:"class"`
	Domain   string    `json:"domain"`
	Password string    `json:"password"`
	Host     string    `json:"host"`
}

// JSON "checker" element
type Checker struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Last   string `json:"last"`
}

// JSON "log" element
type Log struct {
	File string `json:"file"`
}

// JSON configuration file layout
type Config struct {
	Targets []Target `json:"targets"`
	Checker Checker  `json:"checker"`
	Log     Log      `json:"log"`
}

// Specification: https://www.value-domain.com/ddns.php?action=howto
type ValueDomainStatus int

const (
	Success                 ValueDomainStatus = 0
	InvalidRequest          ValueDomainStatus = 1
	InvalidDomainOrPassword ValueDomainStatus = 2
	InvalidIP               ValueDomainStatus = 3
	AuthenticationFailed    ValueDomainStatus = 4
	DatabaseBusy            ValueDomainStatus = 5
	Unknown                 ValueDomainStatus = 9
	ClientError             ValueDomainStatus = -1
)

func main() {
	configFile := flag.String("c", "ddns.json", "path to configuration file")
	flag.Parse()

	// Load config
	config, err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("[Config] Failed to load %s: %v\n", *configFile, err)
	}
	setupLogFile(config.Log)

	// Load last IP
	last, err := loadLastIP(config.Checker.Last)
	if err != nil {
		log.Fatalf("[Last] Failed to load %s: %v\n", config.Checker.Last, err)
	}

	// Check IP
	ip, err := checkIP(config.Checker)
	if err != nil {
		log.Fatalf("[Checker] Failed to check: %v\n", err)
	}

	// Detect change
	if last == ip {
		fmt.Printf("[Checker] IP not changed: %s\n", ip) // console write only
		return
	}

	// Submit IP for each targets
	success := make([]bool, len(config.Targets))
	for i, target := range config.Targets {
		switch target.Class {
		case ValueDomain:
			success[i] = handleValueDomain(target, ip)
		case Cloudflare:
			success[i] = handleCloudflare(target, ip)
		}
	}

	// Don't save if failed result included
	for _, entry := range success {
		if !entry {
			os.Exit(1)
		}
	}

	// Save last successful IP
	if err := writeLastIP(config.Checker.Last, ip); err != nil {
		log.Fatalf("[Last] Failed to write %s: %v\n", config.Checker.Last, err)
	}
}

func loadConfig(path string) (*Config, error) {
	configData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	err = json.Unmarshal(configData, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func setupLogFile(logConfig Log) {
	f, err := os.OpenFile(logConfig.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(io.MultiWriter(os.Stdout, f))
}

func checkIP(checker Checker) (string, error) {
	res, err := http.Get(checker.URL)
	if err != nil {
		return "", err
	}
	if res.StatusCode != 200 {
		return "", errors.New(fmt.Sprintf("[Checker] HTTP %d: %s", res.StatusCode, res.Status))
	}
	rawBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	ipStr := strings.TrimSuffix(string(rawBody), "\n")
	if ip := net.ParseIP(ipStr); ip == nil {
		return "", errors.New("[Checker] Illegal response: " + ipStr)
	}
	return ipStr, nil
}

func loadLastIP(lastFilePath string) (string, error) {
	_, err := os.Stat(lastFilePath)
	if err != nil {
		return "", nil // empty string if file isn't exists
	}
	file, err := os.Open(lastFilePath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("[Last] Failed to close %s: %v", lastFilePath, err)
		}
	}()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func writeLastIP(lastFilePath string, ip string) error {
	_, err := os.Stat(lastFilePath)
	if err == nil {
		if err := os.Remove(lastFilePath); err != nil {
			log.Printf("[Last] Failed to remove %s: %v", lastFilePath, err)
		}
	}
	return ioutil.WriteFile(lastFilePath, []byte(ip), 0644)
}
