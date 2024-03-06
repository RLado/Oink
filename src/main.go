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
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type Config struct {
	Global  GlobConfig
	Domains []DomConfig
}

type GlobConfig struct {
	Secretapikey string
	Apikey       string
	Interval     int
	Ttl          int
}

type DomConfig struct {
	Secretapikey string
	Apikey       string
	Domain       string
	Subdomain    string
	Ttl          int
}

type IP struct {
	Ip    string
	IpVer string
}

// Get the current IP address
// Requests the IP address from the porkbun API & checks if the API keys are valid
func getIp(config DomConfig) (IP, error) {
	ip := IP{}

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	// Prepare request body
	jsonValue, err := json.Marshal(map[string]string{
		"secretapikey": config.Secretapikey,
		"apikey":       config.Apikey,
	})
	if err != nil {
		return ip, fmt.Errorf("error building request body: %s", err)
	}

	// Send API request
	resp, err := client.Post("https://porkbun.com/api/json/v3/ping", "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return ip, fmt.Errorf("error sending API request: %s", err)
	}
	defer resp.Body.Close()

	// Parse API response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ip, fmt.Errorf("error reading API response: %s", err)
	}

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return ip, fmt.Errorf("error decoding API response: %s", err)
	}

	// Use the response
	if data["status"].(string) != "SUCCESS" {
		return ip, fmt.Errorf("error: %s", data["message"].(string))
	}
	ip.Ip = data["yourIp"].(string)

	// Read whether the IP address is IPv4 or IPv6
	if net.ParseIP(ip.Ip).To4() != nil {
		ip.IpVer = "ipv4"
	} else if net.ParseIP(ip.Ip).To16() != nil {
		ip.IpVer = "ipv6"
	} else {
		return ip, fmt.Errorf("error parsing IP address: %s", ip.Ip)
	}

	return ip, nil
}

// Get the current IPv4 address
// Requests the IP address from the porkbun API & checks if the API keys are valid
func getIp4(config DomConfig) (IP, error) {
	ip := IP{}

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	// Prepare request body
	jsonValue, err := json.Marshal(map[string]string{
		"secretapikey": config.Secretapikey,
		"apikey":       config.Apikey,
	})
	if err != nil {
		return ip, fmt.Errorf("error building request body: %s", err)
	}

	// Send API request
	resp, err := client.Post("https://api-ipv4.porkbun.com/api/json/v3/ping", "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return ip, fmt.Errorf("error sending API request: %s", err)
	}
	defer resp.Body.Close()

	// Parse API response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ip, fmt.Errorf("error reading API response: %s", err)
	}

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return ip, fmt.Errorf("error decoding API response: %s", err)
	}

	// Use the response
	if data["status"].(string) != "SUCCESS" {
		return ip, fmt.Errorf("error: %s", data["message"].(string))
	}
	ip.Ip = data["yourIp"].(string)

	// Read whether the IP address is IPv4 or IPv6 (should be IPv4)
	if net.ParseIP(ip.Ip).To4() != nil {
		ip.IpVer = "ipv4"
	} else if net.ParseIP(ip.Ip).To16() != nil {
		ip.IpVer = "ipv6"
	} else {
		return ip, fmt.Errorf("error parsing IP address: %s", ip.Ip)
	}

	return ip, nil
}

// Update the DNS record
// Updates the DNS record with the current IP address
// Returns true if the record was updated, false if it wasn't
func updateDns(config DomConfig, ip IP) (bool, error) {
	client := http.Client{
		Timeout: 30 * time.Second,
	}

	// Prepare request body
	jsonValue, err := json.Marshal(map[string]string{
		"secretapikey": config.Secretapikey,
		"apikey":       config.Apikey,
	})
	if err != nil {
		return false, fmt.Errorf("error building request body: %s", err)
	}

	var recordType string
	if ip.IpVer == "ipv4" {
		recordType = "A"
	} else if ip.IpVer == "ipv6" {
		recordType = "AAAA"
	}

	// Send API request
	resp, err := client.Post(fmt.Sprintf("https://porkbun.com/api/json/v3/dns/retrieveByNameType/%s/%s/%s", config.Domain, recordType, config.Subdomain), "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return false, fmt.Errorf("error sending API request: %s", err)
	}
	defer resp.Body.Close()

	// Parse API response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("error reading API response: %s", err)
	}

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return false, fmt.Errorf("error decoding API response: %s", err)
	}

	// Use the response
	if data["status"].(string) != "SUCCESS" {
		return false, fmt.Errorf("error: %s", data["message"].(string))
	}

	// Check if the record needs to be updated
	var updateReq bool
	var recordId string
	if len(data["records"].([]interface{})) == 0 { // No records found. Create a new one
		// Create a new record
		return createRecord(config, ip)
	} else if len(data["records"].([]interface{})) == 1 { // One record is found. Update if required
		if data["records"].([]interface{})[0].(map[string]interface{})["content"].(string) != ip.Ip {
			// Update the record
			updateReq = true
			// Save the record ID
			recordId = data["records"].([]interface{})[0].(map[string]interface{})["id"].(string)
		}
	} else if len(data["records"].([]interface{})) > 1 { // Multiple records found. Avoid updating
		log.Printf("Warning: Multiple records found for %s.%s -- Not updating any records", config.Subdomain, config.Domain)
	}

	// Update the record
	if !updateReq {
		return false, nil
	}

	// Prepare request body
	jsonValue, err = json.Marshal(map[string]string{
		"secretapikey": config.Secretapikey,
		"apikey":       config.Apikey,
		"name":         config.Subdomain,
		"type":         recordType,
		"content":      ip.Ip,
		"ttl":          fmt.Sprint(config.Ttl),
	})
	if err != nil {
		return false, fmt.Errorf("error building request body: %s", err)
	}

	// Send API request
	resp, err = client.Post(fmt.Sprintf("https://porkbun.com/api/json/v3/dns/edit/%s/%s", config.Domain, recordId), "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return false, fmt.Errorf("error sending API request: %s", err)
	}
	defer resp.Body.Close()

	// Parse API response
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("error reading API response: %s", err)
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return false, fmt.Errorf("error decoding API response: %s", err)
	}

	// Use the response
	if data["status"].(string) != "SUCCESS" {
		return false, fmt.Errorf("error: %s", data["message"].(string))
	}

	return true, nil
}

// Create a new DNS record
// Creates a new DNS record with the current IP address
func createRecord(config DomConfig, ip IP) (bool, error) {
	client := http.Client{
		Timeout: 30 * time.Second,
	}

	// Prepare request body
	var recordType string
	if ip.IpVer == "ipv4" {
		recordType = "A"
	} else if ip.IpVer == "ipv6" {
		recordType = "AAAA"
	}

	jsonValue, err := json.Marshal(map[string]string{
		"secretapikey": config.Secretapikey,
		"apikey":       config.Apikey,
		"name":         config.Subdomain,
		"type":         recordType,
		"content":      ip.Ip,
		"ttl":          fmt.Sprint(config.Ttl),
	})
	if err != nil {
		return false, fmt.Errorf("error building request body: %s", err)
	}

	// Send API request
	resp, err := client.Post(fmt.Sprintf("https://porkbun.com/api/json/v3/dns/create/%s", config.Domain), "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return false, fmt.Errorf("error sending API request: %s", err)
	}
	defer resp.Body.Close()

	// Parse API response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("error reading API response: %s", err)
	}

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return false, fmt.Errorf("error decoding API response: %s", err)
	}

	// Use the response
	if data["status"].(string) != "SUCCESS" {
		return false, fmt.Errorf("error: %s", data["message"].(string))
	}
	log.Printf("Record created successfully with ID: %d", int(data["id"].(float64)))

	return true, nil
}

func main() {
	// Flags
	configPath := flag.String("c", "/etc/oink_ddns/config.json", "Path to oink_ddns configuration file")
	verbose := flag.Bool("v", false, "Enable verbose output")

	flag.Parse()

	// Parse config file
	file, err := os.Open(*configPath)
	if err != nil {
		log.Fatalf("Error opening config file: %s", err)
	}
	defer file.Close()

	jsonDecoder := json.NewDecoder(file)
	config := Config{}
	err = jsonDecoder.Decode(&config)
	if err != nil {
		log.Fatalf("Error decoding config file: %s", err)
	}

	// Enforce minimum interval of 60 seconds
	if config.Global.Interval < 60 {
		if *verbose {
			log.Printf("Warning: Minimum interval is 60 seconds, setting interval to 60 seconds")
		}
		config.Global.Interval = 60
	}

	// Run the update loop
	for {
		// Update domains
		for _, domConfig := range config.Domains {
			// Fill in missing values from the global config
			if domConfig.Secretapikey == "" {
				domConfig.Secretapikey = config.Global.Secretapikey
			}
			if domConfig.Apikey == "" {
				domConfig.Apikey = config.Global.Apikey
			}
			if domConfig.Ttl == 0 {
				domConfig.Ttl = config.Global.Ttl
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

		// Wait for the next update
		if *verbose {
			log.Printf("Waiting %d seconds for the next update", config.Global.Interval)
		}
		time.Sleep(time.Duration(config.Global.Interval) * time.Second)
	}
}
