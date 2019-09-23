// Copyright 2017-2019 mikan.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

const (
	cfLogPrefix     = "[Submitter/Cloudflare] "
	cfZoneIDParam   = "{zone_id}"
	cfRecordIDParam = "{record_id}"
	cfNameParam     = "{name}"
	cfEndpoint      = "https://api.cloudflare.com/client/v4/"
	cfQueryPath     = cfEndpoint + "zones/" + cfZoneIDParam + "/dns_records?type=A&name=" + cfNameParam
	cfRecordPath    = cfEndpoint + "zones/" + cfZoneIDParam + "/dns_records/" + cfRecordIDParam
)

type cfQueryResponse struct {
	Result  []cfQueryResult `json:"result"`
	Success bool            `json:"success"`
}

type cfUpdateResponse struct {
	Success bool `json:"success"`
}

type cfQueryResult struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

type cfRecordUpdateRequest struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	Proxied bool   `json:"proxied"`
}

func handleCloudflare(target Target, ip string) bool {
	recordID, err := findRecordID(target)
	if err != nil {
		log.Printf(cfLogPrefix+"Failed to find record ID %s: %v\n", target.Domain, err)
		return false
	}
	if err := submitCloudflare(target, recordID, ip); err != nil {
		log.Printf(cfLogPrefix+"Failed to submit %s: %v\n", target.Class, err)
		return false
	}
	log.Printf(cfLogPrefix+"Success %s.%s=%s\n", target.Host, target.Domain, ip)
	return true
}

func findRecordID(target Target) (string, error) {
	url := strings.Replace(cfQueryPath, cfZoneIDParam, target.Domain, 1)
	url = strings.Replace(url, cfNameParam, target.Host, 1)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build query request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+target.Password)
	client := new(http.Client)
	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send query request: %v", err)
	}
	if res.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d %s", res.StatusCode, res.Status)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Printf(cfLogPrefix+"Failed to close body: %v", err)
		}
	}()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read body: %v", err)
	}
	var queryResponse cfQueryResponse
	if err := json.Unmarshal(body, &queryResponse); err != nil {
		return "", fmt.Errorf("failed to unmarshal body: %v", err)
	}
	if !queryResponse.Success {
		return "", fmt.Errorf("query failed with unsuccess result: %v", string(body))
	}
	if len(queryResponse.Result) < 1 {
		return "", fmt.Errorf("query failed with empty result: %v", string(body))
	}
	fmt.Printf(cfLogPrefix+"Found record: %s\n", queryResponse.Result[0].ID)
	return queryResponse.Result[0].ID, nil
}

func submitCloudflare(target Target, recordID, ip string) error {
	url := strings.Replace(cfRecordPath, cfZoneIDParam, target.Domain, 1)
	url = strings.Replace(url, cfRecordIDParam, recordID, 1)
	requestBody, err := json.Marshal(cfRecordUpdateRequest{
		Type:    "A",
		Name:    target.Host,
		Content: ip,
		Proxied: target.Proxied,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal update request: %v", err)
	}
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("failed to build update request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+target.Password)
	client := new(http.Client)
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send update request: %v", err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Printf(cfLogPrefix+"Failed to close body: %v", err)
		}
	}()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read update response: %v", err)
	}
	var putResponse cfUpdateResponse
	if err := json.Unmarshal(body, &putResponse); err != nil {
		return fmt.Errorf("failed to parse respones: %v", err)
	}
	if !putResponse.Success {
		return fmt.Errorf("failed to put with unsuccess result: %v", string(body))
	}
	return nil
}
