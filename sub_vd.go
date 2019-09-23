// Copyright 2017-2019 mikan.
package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const vdLogPrefix = "[Submitter/ValueDomain] "

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

func handleValueDomain(target Target, ip string) bool {
	status, err := submitValueDomain(target, ip)
	if err != nil {
		log.Printf(vdLogPrefix+"Failed to submit %s: %v\n", target.Class, err)
		return false
	}
	switch status {
	case Success:
		log.Printf(vdLogPrefix+"Success %s.%s=%s\n", target.Host, target.Domain, ip)
		return true
	case InvalidRequest:
		log.Printf(vdLogPrefix+"INVALID REQUEST %s.%s=%s\n", target.Host, target.Domain, ip)
	case InvalidDomainOrPassword:
		log.Printf(vdLogPrefix+"INVALID DOMAIN OR PASSWORD %s.%s=%s\n", target.Host, target.Domain, ip)
	case InvalidIP:
		log.Printf(vdLogPrefix+"INVALID IP %s.%s=%s\n", target.Host, target.Domain, ip)
	case AuthenticationFailed:
		log.Printf(vdLogPrefix+"AUTHENTICATION FAILED %s.%s=%s\n", target.Host, target.Domain, ip)
	case DatabaseBusy:
		log.Printf(vdLogPrefix+"DATABASE BUSY %s.%s=%s\n", target.Host, target.Domain, ip)
	case Unknown:
		fallthrough
	default:
		log.Printf(vdLogPrefix+"Unknown %s.%s=%s\n", target.Host, target.Domain, ip)
	}
	return false
}

func submitValueDomain(target Target, ip string) (ValueDomainStatus, error) {
	url := fmt.Sprintf("https://dyn.value-domain.com/cgi-bin/dyn.fcg?d=%s&p=%s&h=%s&i=%s",
		target.Domain, target.Password, target.Host, ip)
	res, err := http.Get(url)
	if err != nil {
		return ClientError, err
	}
	if res.StatusCode != 200 {
		return Unknown, fmt.Errorf("HTTP %d %s", res.StatusCode, res.Status)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Printf(vdLogPrefix+"Failed to close body: %v", err)
		}
	}()
	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "status=") {
			status, err := strconv.Atoi(strings.Replace(strings.TrimSpace(line), "status=", "", 1))
			if err != nil {
				return ClientError, fmt.Errorf("failed to detect status: %s", line)
			}
			fmt.Println(vdLogPrefix + line) // console write only
			return ValueDomainStatus(status), nil
		}
	}
	return ClientError, fmt.Errorf("received HTTP %d but status not detected", res.StatusCode)
}
