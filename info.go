package main

type ClientInfo struct {
	HostID       string   `json:"host_id"`
	Hostname     string   `json:"hostname"`
	Username     string   `json:"username"`
	OS           string   `json:"os"`
	CPU          string   `json:"cpu"`
	Memory       string   `json:"memory"`
	Disk         string   `json:"disk"`
	IPAddresses  []string `json:"ip_addresses"`
	MACAddresses []string `json:"mac_addresses"`
	Programs     []string `json:"programs"`
	Updated      string   `json:"updated"`
}
