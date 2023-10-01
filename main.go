/*
* Oink!
* A lightweight DDNS client for porkbun.com
*
* Author: Ricard Lado <ricard@lado.one>
* Repository:
*
* Version: 0.1
* License: MIT
 */

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type Config struct {
	Secretapikey string
	Apikey       string
	Domain       string
	Subdomain    string
	Ttl          int
	Priority     int
	Interval     int
}

type IP struct {
	Ip    string
	IpVer string
}

// Get the current IP address
// Requests the IP address from the porkbun API & checks if the API keys are valid
func get_ip(config Config) (IP, error) {
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
	body, err := ioutil.ReadAll(resp.Body)
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

// Update the DNS record
// Updates the DNS record with the current IP address
// Returns true if the record was updated, false if it wasn't
func update_dns(config Config, ip IP) (bool, error) {
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
	body, err := ioutil.ReadAll(resp.Body)
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
	var update_req bool
	var record_id string
	if len(data["records"].([]interface{})) == 0 { // No records found. Create a new one
		// Create a new record
		return create_record(config, ip)
	} else if len(data["records"].([]interface{})) == 1 { // One record is found. Update if required
		if data["records"].([]interface{})[0].(map[string]interface{})["content"].(string) != ip.Ip {
			// Update the record
			update_req = true
			// Save the record ID
			record_id = data["records"].([]interface{})[0].(map[string]interface{})["id"].(string)
		}
	} else if len(data["records"].([]interface{})) > 1 { // Multiple records found. Update if a record with a "ddns" annotation is found
		log.Printf("Warning: Multiple records found, make sure to add a \"ddns\" annotation to the record you wish to update")
		annotation_found := false

		// Look for a record with a notes value of "ddns"
		for _, record := range data["records"].([]interface{}) {
			if record.(map[string]interface{})["notes"].(string) == "ddns" {
				annotation_found = true
				// Check if the record needs to be updated
				if record.(map[string]interface{})["content"].(string) != ip.Ip {
					// Update the record
					update_req = true
					// Save the record ID
					record_id = record.(map[string]interface{})["id"].(string)
				}
			}
		}
		if !annotation_found {
			log.Fatal("Error: Multiple records found, but none with a \"ddns\" annotation")
		}
	}

	// Update the record
	if !update_req {
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
		"prio":         fmt.Sprint(config.Priority),
	})
	if err != nil {
		return false, fmt.Errorf("error building request body: %s", err)
	}

	// Send API request
	resp, err = client.Post(fmt.Sprintf("https://porkbun.com/api/json/v3/dns/edit/%s/%s", config.Domain, record_id), "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return false, fmt.Errorf("error sending API request: %s", err)
	}
	defer resp.Body.Close()

	// Parse API response
	body, err = ioutil.ReadAll(resp.Body)
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
func create_record(config Config, ip IP) (bool, error) {
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
		"prio":         fmt.Sprint(config.Priority),
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
	body, err := ioutil.ReadAll(resp.Body)
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
	config_path := flag.String("c", "/etc/oink_ddns/config.json", "Path to oink_ddns configuration file")
	verbose := flag.Bool("v", false, "Enable verbose output")

	flag.Parse()

	// Parse config file
	file, err := os.Open(*config_path)
	if err != nil {
		log.Fatalf("Error opening config file: %s", err)
	}
	defer file.Close()

	json_decoder := json.NewDecoder(file)
	config := Config{}
	err = json_decoder.Decode(&config)
	if err != nil {
		log.Fatalf("Error decoding config file: %s", err)
	}

	// Run the update loop
	for {
		if *verbose {
			log.Printf("Updating record: %s.%s", config.Subdomain, config.Domain)
		}
		// Get current IP address
		current_ip, err := get_ip(config)
		if err != nil {
			log.Fatalln(err)
		}
		if *verbose {
			log.Printf("Current IP address: %s", current_ip.Ip)
		}

		// Update DNS record
		updated, err := update_dns(config, current_ip)
		if err != nil {
			log.Fatalln(err)
		}
		if updated {
			log.Printf("Record updated successfully to: %s", current_ip.Ip)
		} else if *verbose {
			log.Printf("Record is already up to date")
		}

		// Wait for the next update
		if *verbose {
			log.Printf("Waiting %d seconds for the next update", config.Interval)
		}
		time.Sleep(time.Duration(config.Interval) * time.Second)
	}
}
