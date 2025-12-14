package util

import (
	"net"
	"strings"
)

// IPVerifier checks if an IP address is allowed
type IPVerifier struct {
	whitelist []string
}

// NewIPVerifier creates a new IP verifier
func NewIPVerifier(whitelist []string) *IPVerifier {
	return &IPVerifier{
		whitelist: whitelist,
	}
}

// IsAllowed checks if an IP address is allowed
func (v *IPVerifier) IsAllowed(ipAddress string, logger *Logger) bool {
	// If whitelist contains *, allow all
	for _, allowed := range v.whitelist {
		if allowed == "*" {
			return true
		}
	}

	// Extract IP from address (remove port if present)
	host, _, err := net.SplitHostPort(ipAddress)
	if err != nil {
		// If no port, use the address as-is
		host = ipAddress
	}

	// Check if IP is in whitelist
	for _, allowed := range v.whitelist {
		if matchesPattern(host, allowed) {
			return true
		}
	}

	logger.Warnf("Blocking request from %s", ipAddress)
	return false
}

// matchesPattern checks if an IP matches a pattern
func matchesPattern(ip, pattern string) bool {
	// Exact match
	if ip == pattern {
		return true
	}

	// CIDR notation
	if strings.Contains(pattern, "/") {
		_, network, err := net.ParseCIDR(pattern)
		if err != nil {
			return false
		}
		
		ipAddr := net.ParseIP(ip)
		if ipAddr == nil {
			return false
		}
		
		return network.Contains(ipAddr)
	}

	// Wildcard pattern (e.g., 192.168.*.*)
	if strings.Contains(pattern, "*") {
		ipParts := strings.Split(ip, ".")
		patternParts := strings.Split(pattern, ".")
		
		if len(ipParts) != len(patternParts) {
			return false
		}
		
		for i := range ipParts {
			if patternParts[i] != "*" && ipParts[i] != patternParts[i] {
				return false
			}
		}
		
		return true
	}

	return false
}
