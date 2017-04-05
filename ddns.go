// Copyright 2017 mikan.
package main

import (
	"bufio"
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
	"strconv"
	"strings"
)

// JSON "target" element
type Target struct {
	Class    string `json:"class"`
	Domain   string `json:"domain"`
	Password string `json:"password"`
	Host     string `json:"host"`
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
	SUCCESS                    ValueDomainStatus = 0
	INVALID_REQUEST            ValueDomainStatus = 1
	INVALID_DOMAIN_OR_PASSWORD ValueDomainStatus = 2
	INVALID_IP                 ValueDomainStatus = 3
	AUTHENTICATION_FAILED      ValueDomainStatus = 4
	DATABASE_BUSY              ValueDomainStatus = 5
	UNKNOWN                    ValueDomainStatus = 9
	CLIENT_ERROR               ValueDomainStatus = -1
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
	if *last == *ip {
		fmt.Printf("[Checker] IP not changed: %s\n", *ip) // console write only
		return
	}

	// Submit IP for each targets
	success := false
	for _, target := range config.Targets {
		status, err := submit(target, *ip)
		if err != nil {
			log.Printf("[Submitter] Failed to submit %s: %v\n", target.Class, err)
			continue
		}
		switch status {
		case SUCCESS:
			log.Printf("[Submitter] SUCCESS %s.%s=%s\n", target.Host, target.Domain, *ip)
			success = true
		case INVALID_REQUEST:
			log.Printf("[Submitter] INVALID REQUEST %s.%s=%s\n", target.Host, target.Domain, *ip)
		case INVALID_DOMAIN_OR_PASSWORD:
			log.Printf("[Submitter] INVALID DOMAIN OR PASSWORD %s.%s=%s\n", target.Host, target.Domain, *ip)
		case INVALID_IP:
			log.Printf("[Submitter] INVALID IP %s.%s=%s\n", target.Host, target.Domain, *ip)
		case AUTHENTICATION_FAILED:
			log.Printf("[Submitter] AUTHENTICATION FAILED %s.%s=%s\n", target.Host, target.Domain, *ip)
		case DATABASE_BUSY:
			log.Printf("[Submitter] DATABASE BUSY %s.%s=%s\n", target.Host, target.Domain, *ip)
		case UNKNOWN:
			fallthrough
		default:
			log.Printf("[Submitter] UNKNOWN %s.%s=%s\n", target.Host, target.Domain, *ip)
		}
	}

	// Save last successful IP
	if !success {
		os.Exit(1)
	}
	err = writeLastIP(config.Checker.Last, *ip)
	if err != nil {
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

func checkIP(checker Checker) (*string, error) {
	res, err := http.Get(checker.URL)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("[Checker] HTTP %d: %s", res.StatusCode, res.Status))
	}
	rawBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	body := string(rawBody)
	ip := net.ParseIP(body)
	if ip == nil {
		return nil, errors.New("[Checker] Illegal response: " + body)
	}
	return &body, nil
}

func loadLastIP(lastFilePath string) (*string, error) {
	_, err := os.Stat(lastFilePath)
	if err != nil {
		empty := ""
		return &empty, nil // empty string if file isn't exists
	}
	file, err := os.Open(lastFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	last := string(data)
	return &last, nil
}

func writeLastIP(lastFilePath string, ip string) error {
	_, err := os.Stat(lastFilePath)
	if err == nil {
		os.Remove(lastFilePath)
	}
	return ioutil.WriteFile(lastFilePath, []byte(ip), 0644)
}

func submit(target Target, ip string) (ValueDomainStatus, error) {
	url := fmt.Sprintf("https://dyn.value-domain.com/cgi-bin/dyn.fcg?d=%s&p=%s&h=%s&i=%s",
		target.Domain, target.Password, target.Host, ip)
	res, err := http.Get(url)
	if err != nil {
		return CLIENT_ERROR, err
	}
	if res.StatusCode != 200 {
		return UNKNOWN, errors.New("HTTP " + strconv.Itoa(res.StatusCode) + " " + res.Status)
	}
	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "status=") {
			status, err := strconv.Atoi(strings.Replace(strings.TrimSpace(line), "status=", "", 1))
			if err != nil {
				return CLIENT_ERROR, errors.New("Failed to detect status: " + line)
			}
			fmt.Println("[Submitter] " + line) // console write only
			return ValueDomainStatus(status), nil
		}
	}
	return CLIENT_ERROR, errors.New("HTTP 200 received but status not detected.")
}
