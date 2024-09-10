/*
* Oink!
* A lightweight DDNS client for porkbun.com
*
* Author: Ricard Lado <ricard@lado.one>
* Repository: https://github.com/RLado/Oink
*
* License: MIT
 */

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type config struct {
	Global  globConfig  `json:"global"`
	Domains []domConfig `json:"domains"`
}

type globConfig struct {
	Secretapikey string `json:"secretapikey"`
	Apikey       string `json:"apikey"`
	Interval     int    `json:"interval"`
	Ttl          int    `json:"ttl"`
}

type domConfig struct {
	Secretapikey string `json:"secretapikey"`
	Apikey       string `json:"apikey"`
	Domain       string `json:"domain"`
	Subdomain    string `json:"subdomain"`
	Ttl          int    `json:"ttl"`
}

type ip struct {
	Ip    string
	IpVer string
}

// Response types
type statusResp struct {
	Status  string      `json:"status"`
	Id      json.Number `json:"id"`
	Message string      `json:"message"`
}

type pingResp struct {
	Status  string `json:"status"`
	Ip      string `json:"yourIp"`
	Message string `json:"message"`
}

type dnsResp struct {
	Status  string   `json:"status"`
	Records []record `json:"records"`
	Message string   `json:"message"`
}

type record struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	Ttl     string `json:"ttl"`
	Prio    string `json:"prio"`
	Notes   string `json:"notes"`
}

// Request types
type authReq struct {
	Secretapikey string `json:"secretapikey"`
	Apikey       string `json:"apikey"`
}

type recordReq struct {
	Secretapikey string `json:"secretapikey"`
	Apikey       string `json:"apikey"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Content      string `json:"content"`
	Ttl          string `json:"ttl"`
}

// Get the current IP address
// Requests the IP address from the porkbun API & checks if the API keys are valid
func getIp(cfg domConfig) (ip, error) {
	ipAddr := ip{}

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	// Prepare request body
	reqBody, err := json.Marshal(authReq{
		Secretapikey: cfg.Secretapikey,
		Apikey:       cfg.Apikey,
	})
	if err != nil {
		return ipAddr, fmt.Errorf("error building request body: %s", err)
	}

	// Send API request
	resp, err := client.Post("https://porkbun.com/api/json/v3/ping", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return ipAddr, fmt.Errorf("error sending API request: %s", err)
	}
	defer resp.Body.Close()

	// Parse API response
	var data pingResp
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return ipAddr, fmt.Errorf("error decoding API response: %s (status: %d)", err, resp.StatusCode)
	}

	// Use the response
	if data.Status != "SUCCESS" {
		return ipAddr, fmt.Errorf("error %d: %s", resp.StatusCode, data.Message)
	}
	ipAddr.Ip = data.Ip

	// Read whether the IP address is IPv4 or IPv6
	if net.ParseIP(ipAddr.Ip).To4() != nil {
		ipAddr.IpVer = "ipv4"
	} else if net.ParseIP(ipAddr.Ip).To16() != nil {
		ipAddr.IpVer = "ipv6"
	} else {
		return ipAddr, fmt.Errorf("error parsing IP address: %s", ipAddr.Ip)
	}

	return ipAddr, nil
}

// Get the current IPv4 address
// Requests the IP address from the porkbun API & checks if the API keys are valid
func getIp4(cfg domConfig) (ip, error) {
	ipAddr := ip{}

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	// Prepare request body
	reqBody, err := json.Marshal(authReq{
		Secretapikey: cfg.Secretapikey,
		Apikey:       cfg.Apikey,
	})
	if err != nil {
		return ipAddr, fmt.Errorf("error building request body: %s", err)
	}

	// Send API request
	resp, err := client.Post("https://api-ipv4.porkbun.com/api/json/v3/ping", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return ipAddr, fmt.Errorf("error sending API request: %s", err)
	}
	defer resp.Body.Close()

	// Parse API response
	var data pingResp
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return ipAddr, fmt.Errorf("error decoding API response: %s (status: %d)", err, resp.StatusCode)
	}

	// Use the response
	if data.Status != "SUCCESS" {
		return ipAddr, fmt.Errorf("error %d: %s", resp.StatusCode, data.Message)
	}
	ipAddr.Ip = data.Ip

	// Read whether the IP address is IPv4 or IPv6 (should be IPv4)
	if net.ParseIP(ipAddr.Ip).To4() != nil {
		ipAddr.IpVer = "ipv4"
	} else if net.ParseIP(ipAddr.Ip).To16() != nil {
		ipAddr.IpVer = "ipv6"
	} else {
		return ipAddr, fmt.Errorf("error parsing IP address: %s", ipAddr.Ip)
	}

	return ipAddr, nil
}

// Update the DNS record
// Updates the DNS record with the current IP address
// Returns true if the record was updated, false if it wasn't
func updateDns(cfg domConfig, ipAddr ip) (bool, error) {
	client := http.Client{
		Timeout: 30 * time.Second,
	}

	// Prepare request body
	reqBody, err := json.Marshal(authReq{
		Secretapikey: cfg.Secretapikey,
		Apikey:       cfg.Apikey,
	})
	if err != nil {
		return false, fmt.Errorf("error building request body: %s", err)
	}

	var recordType string
	if ipAddr.IpVer == "ipv4" {
		recordType = "A"
	} else if ipAddr.IpVer == "ipv6" {
		recordType = "AAAA"
	}

	// Send API request
	resp, err := client.Post(fmt.Sprintf("https://porkbun.com/api/json/v3/dns/retrieveByNameType/%s/%s/%s", cfg.Domain, recordType, cfg.Subdomain), "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return false, fmt.Errorf("error sending API request: %s", err)
	}
	defer resp.Body.Close()

	// Parse API response
	var data dnsResp
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return false, fmt.Errorf("error decoding API response: %s (status: %d)", err, resp.StatusCode)
	}

	// Use the response
	if data.Status != "SUCCESS" {
		return false, fmt.Errorf("error %d: %s", resp.StatusCode, data.Message)
	}

	// Check if the record needs to be updated
	var updateReq bool
	var recordId string
	if len(data.Records) == 0 { // No records found. Create a new one
		// Create a new record
		return createRecord(cfg, ipAddr)
	} else if len(data.Records) == 1 { // One record is found. Update if required
		if data.Records[0].Content != ipAddr.Ip {
			// Update the record
			updateReq = true
			// Save the record ID
			recordId = data.Records[0].Id
		}
	} else if len(data.Records) > 1 { // Multiple records found. Avoid updating
		log.Printf("Warning: Multiple records found for %s.%s -- Not updating any records", cfg.Subdomain, cfg.Domain)
	}

	// Update the record
	if !updateReq {
		return false, nil
	}

	// Prepare request body
	reqBody, err = json.Marshal(recordReq{
		Secretapikey: cfg.Secretapikey,
		Apikey:       cfg.Apikey,
		Name:         cfg.Subdomain,
		Type:         recordType,
		Content:      ipAddr.Ip,
		Ttl:          fmt.Sprint(cfg.Ttl),
	})
	if err != nil {
		return false, fmt.Errorf("error building request body: %s", err)
	}

	// Send API request
	resp, err = client.Post(fmt.Sprintf("https://porkbun.com/api/json/v3/dns/edit/%s/%s", cfg.Domain, recordId), "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return false, fmt.Errorf("error sending API request: %s", err)
	}
	defer resp.Body.Close()

	// Parse API response
	var status statusResp
	err = json.NewDecoder(resp.Body).Decode(&status)
	if err != nil {
		return false, fmt.Errorf("error decoding API response: %s (status: %d)", err, resp.StatusCode)
	}

	// Use the response
	if status.Status != "SUCCESS" {
		return false, fmt.Errorf("error %d: %s", resp.StatusCode, status.Message)
	}

	return true, nil
}

// Create a new DNS record
// Creates a new DNS record with the current IP address
func createRecord(cfg domConfig, ipAddr ip) (bool, error) {
	client := http.Client{
		Timeout: 30 * time.Second,
	}

	// Prepare request body
	var recordType string
	if ipAddr.IpVer == "ipv4" {
		recordType = "A"
	} else if ipAddr.IpVer == "ipv6" {
		recordType = "AAAA"
	}

	req, err := json.Marshal(recordReq{
		Secretapikey: cfg.Secretapikey,
		Apikey:       cfg.Apikey,
		Name:         cfg.Subdomain,
		Type:         recordType,
		Content:      ipAddr.Ip,
		Ttl:          fmt.Sprint(cfg.Ttl),
	})
	if err != nil {
		return false, fmt.Errorf("error building request body: %s", err)
	}

	// Send API request
	resp, err := client.Post(fmt.Sprintf("https://porkbun.com/api/json/v3/dns/create/%s", cfg.Domain), "application/json", bytes.NewBuffer(req))
	if err != nil {
		return false, fmt.Errorf("error sending API request: %s", err)
	}
	defer resp.Body.Close()

	// Parse API response
	var status statusResp
	err = json.NewDecoder(resp.Body).Decode(&status)
	if err != nil {
		return false, fmt.Errorf("error decoding API response: %s (status: %d)", err, resp.StatusCode)
	}

	// Use the response
	if status.Status != "SUCCESS" {
		return false, fmt.Errorf("error %d: %s", resp.StatusCode, status.Message)
	}
	log.Printf("Record created successfully with ID: %s", status.Id)

	return true, nil
}

func main() {
	// Flags
	configPath := flag.String("c", "/etc/oink_ddns/config.json", "Path to oink_ddns configuration file")
	update := flag.Bool("u", false, "Update the DNS records immediately and exit")
	verbose := flag.Bool("v", false, "Enable verbose output")

	flag.Parse()

	// Parse config file
	file, err := os.Open(*configPath)
	if err != nil {
		log.Fatalf("Error opening config file: %s", err)
	}

	cfg := config{}
	err = json.NewDecoder(file).Decode(&cfg)
	if err != nil {
		log.Fatalf("Error decoding config file: %s", err)
	}

	file.Close()

	// Enforce minimum interval of 60 seconds
	if cfg.Global.Interval < 60 {
		if *verbose {
			log.Printf("Warning: Minimum interval is 60 seconds, setting interval to 60 seconds")
		}
		cfg.Global.Interval = 60
	}

	// If environment variables for the API keys are set, override the global API keys of the config file
	if os.Getenv("OINK_OVERRIDE_SECRETAPIKEY") != "" && os.Getenv("OINK_OVERRIDE_APIKEY") != "" {
		if *verbose {
			log.Printf("Overriding secretapikey and apikey with environment variables")
		}
		cfg.Global.Secretapikey = os.Getenv("OINK_OVERRIDE_SECRETAPIKEY")
		cfg.Global.Apikey = os.Getenv("OINK_OVERRIDE_APIKEY")
	}

	// Run the update loop
	for {
		// Update domains
		for _, domConfig := range cfg.Domains {
			// Fill in missing values from the global config
			if domConfig.Secretapikey == "" {
				domConfig.Secretapikey = cfg.Global.Secretapikey
			}
			if domConfig.Apikey == "" {
				domConfig.Apikey = cfg.Global.Apikey
			}
			if domConfig.Ttl == 0 {
				domConfig.Ttl = cfg.Global.Ttl
			}

			// Enforce minimum TTL of 600 seconds (as defined by porkbun)
			if domConfig.Ttl < 600 {
				if *verbose {
					log.Printf("Warning: Minimum TTL is 600 seconds, setting TTL for %s.%s to 600 seconds", domConfig.Subdomain, domConfig.Domain)
				}
				domConfig.Ttl = 600
			}

			// Start the update record process
			if *verbose {
				log.Printf("Updating record: %s.%s", domConfig.Subdomain, domConfig.Domain)
			}
			// Get current IP address
			currentIp, err := getIp(domConfig)
			if err != nil {
				log.Fatalln(err)
			}
			if *verbose {
				log.Printf("Current IP address: %s", currentIp.Ip)
			}

			// Update DNS record
			updated, err := updateDns(domConfig, currentIp)
			if err != nil {
				log.Fatalln(err)
			}
			if updated {
				if currentIp.IpVer == "ipv4" {
					log.Printf("A record for %s.%s updated successfully to: %s", domConfig.Subdomain, domConfig.Domain, currentIp.Ip)
				} else if currentIp.IpVer == "ipv6" {
					log.Printf("AAAA record for %s.%s updated successfully to: %s", domConfig.Subdomain, domConfig.Domain, currentIp.Ip)
				}
			} else if *verbose {
				log.Printf("Record is already up to date")
			}

			// If the IP address found was IPv6 check if an IPv4 address is also available
			if currentIp.IpVer == "ipv6" {
				if *verbose {
					log.Printf("IPv6 address found, checking for IPv4")
				}
				ipv4, err := getIp4(domConfig)
				if err != nil {
					if *verbose {
						log.Printf("No IPv4 address found: %s", err)
					}
				} else {
					if *verbose {
						log.Printf("IPv4 address found: %s", ipv4.Ip)
					}

					// Update DNS record
					updated, err := updateDns(domConfig, ipv4)
					if err != nil {
						log.Fatalln(err)
					}
					if updated {
						log.Printf("A record for %s.%s updated successfully to: %s", domConfig.Subdomain, domConfig.Domain, ipv4.Ip)
					} else if *verbose {
						log.Printf("Record is already up to date")
					}
				}
			}
		}

		// Exit if the update flag is set
		if *update {
			os.Exit(0)
		}

		// Wait for the next update
		if *verbose {
			log.Printf("Waiting %d seconds for the next update", cfg.Global.Interval)
		}
		time.Sleep(time.Duration(cfg.Global.Interval) * time.Second)
	}
}
