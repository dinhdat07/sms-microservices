package nft

import (
	"os/exec"
	"strings"
	"sync"
)

var nftMu sync.Mutex

func FlushDownIPs() ([]byte, error) {
	nftMu.Lock()
	defer nftMu.Unlock()
	cmd := exec.Command("nft", "flush", "set", "inet", "sim", "down_ips")
	return cmd.CombinedOutput()
}

func AddDownIP(ip string) ([]byte, error) {
	nftMu.Lock()
	defer nftMu.Unlock()
	cmd := exec.Command("nft", "add", "element", "inet", "sim", "down_ips", "{", ip, "}")
	return cmd.CombinedOutput()
}

func DeleteDownIP(ip string) ([]byte, error) {
	nftMu.Lock()
	defer nftMu.Unlock()
	cmd := exec.Command("nft", "delete", "element", "inet", "sim", "down_ips", "{", ip, "}")
	return cmd.CombinedOutput()
}

func ListDownIPs() (map[string]bool, error) {
	nftMu.Lock()
	defer nftMu.Unlock()
	cmd := exec.Command("nft", "list", "set", "inet", "sim", "down_ips")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	downSet := make(map[string]bool)
	output := string(out)
	if idx := strings.Index(output, "elements = {"); idx != -1 {
		rest := output[idx+len("elements = {"):]
		endIdx := strings.Index(rest, "}")
		if endIdx != -1 {
			elements := strings.TrimSpace(rest[:endIdx])
			if elements != "" {
				for _, raw := range strings.Split(elements, ",") {
					ip := strings.TrimSpace(raw)
					if ip != "" {
						downSet[ip] = true
					}
				}
			}
		}
	}
	return downSet, nil
}
