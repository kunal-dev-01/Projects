package validator

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/mssola/user_agent"
)

const (
	geoIPURL     = "http://ip-api.com/json/%s?fields=status,message,city,country"
	geoIPTimeout = 2 * time.Second
)

type ClientInfo struct {
	OS         string `json:"os"`
	Browser    string `json:"browser"`
	DeviceName string `json:"device_name"`
	City       string `json:"city"`
	Country    string `json:"country"`
}

func (c ClientInfo) IsLocalNetwork() bool {
	return c.City == "Local Network"
}

var ipClient = &http.Client{
	Timeout: geoIPTimeout,
}

func ParseClientInfo(ip, userAgent string) ClientInfo {
	info := ClientInfo{
		OS:         "Unknown",
		Browser:    "Unknown",
		DeviceName: "Unknown",
		City:       "Unknown",
		Country:    "Unknown",
	}

	info.parseUserAgent(userAgent)
	info.parseLocation(ip)

	return info
}

/* ───────────────── Deep Core Extractors ───────────────── */

func (c *ClientInfo) parseUserAgent(uaString string) {
	if uaString == "" {
		return
	}

	ua := user_agent.New(uaString)
	c.OS = ua.OS()

	if name, version := ua.Browser(); name != "" {
		if version != "" {
			c.Browser = name + " " + version
		} else {
			c.Browser = name
		}
	}

	switch {
	case ua.Mobile():
		c.DeviceName = "Mobile"
	case ua.Bot():
		c.DeviceName = "Bot"
	default:
		c.DeviceName = "Desktop"
	}
}

func (c *ClientInfo) parseLocation(ip string) {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return
	}

	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return
	}

	if parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsLinkLocalUnicast() {
		c.City = "Local Network"
		c.Country = "Local Network"
		return
	}

	if loc := fetchGeoIP(ip); loc != nil {
		if loc.City != "" {
			c.City = loc.City
		}
		if loc.Country != "" {
			c.Country = loc.Country
		}
	}
}

/* ───────────────── Non-Blocking Fast Network Lookups ───────────────── */

type geoIPResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	City    string `json:"city"`
	Country string `json:"country"`
}

func fetchGeoIP(ip string) *geoIPResponse {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(geoIPURL, ip), nil)
	if err != nil {
		return nil
	}

	resp, err := ipClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var data geoIPResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil
	}

	return &data
}