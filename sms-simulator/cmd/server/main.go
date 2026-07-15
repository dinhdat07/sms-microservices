package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

var totalIPs int
var subnet string
var nftMu sync.Mutex

type toggleRequest struct {
	IPs []string `json:"ips"`
}

type statusResponse struct {
	Total int `json:"total"`
	Down  int `json:"down"`
}

func main() {
	totalIPs = 10000
	if v := os.Getenv("SIMULATOR_IP_COUNT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			totalIPs = n
		}
	}
	subnet = envStr("SIMULATOR_SUBNET", "10.1")

	mux := http.NewServeMux()
	mux.HandleFunc("/up", handleUp)
	mux.HandleFunc("/down", handleDown)
	mux.HandleFunc("/status", handleStatus)
	mux.HandleFunc("/reset", handleReset)

	if envBool("SIMULATOR_AUTO_FLAP_ENABLED", false) {
		startAutoFlapper()
	}

	log.Printf("Simulator API listening on :8080 (%d IPs, subnet=%s)", totalIPs, subnet)
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if out, err := flushDownIPs(); err != nil {
		log.Printf("nft flush failed: %v, output: %s", err, out)
		http.Error(w, "nft error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleDown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req toggleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	for _, ip := range cleanIPs(req.IPs) {
		if out, err := addDownIP(ip); err != nil {
			log.Printf("nft add failed for %s: %v, output: %s", ip, err, out)
			http.Error(w, "nft error", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req toggleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	for _, ip := range cleanIPs(req.IPs) {
		_, _ = deleteDownIP(ip) // Ignore error if IP was not in set.
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	downSet, err := listDownIPs()
	if err != nil {
		log.Printf("nft list failed: %v", err)
		http.Error(w, "nft error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(statusResponse{
		Total: totalIPs,
		Down:  len(downSet),
	})
}

func startAutoFlapper() {
	interval := envDuration("SIMULATOR_FLAP_INTERVAL", 60*time.Second)
	percent := envFloat("SIMULATOR_FLAP_PERCENT", 5)
	if interval <= 0 || percent <= 0 {
		log.Printf("Auto flapper disabled: interval=%s percent=%.2f", interval, percent)
		return
	}

	ips := generateIPs(totalIPs, subnet)
	flipCount := int(float64(totalIPs) * percent / 100)
	if flipCount < 1 && totalIPs > 0 {
		flipCount = 1
	}
	if flipCount > totalIPs {
		flipCount = totalIPs
	}

	seed := time.Now().UnixNano()
	if v := os.Getenv("SIMULATOR_FLAP_SEED"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			seed = n
		}
	}
	rng := rand.New(rand.NewSource(seed))

	log.Printf("Auto flapper enabled: interval=%s, flip=%.2f%% (%d/%d), seed=%d", interval, percent, flipCount, totalIPs, seed)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if err := flapOnce(ips, flipCount, rng); err != nil {
				log.Printf("auto flap failed: %v", err)
			}
		}
	}()
}

func flapOnce(ips []string, flipCount int, rng *rand.Rand) error {
	downSet, err := listDownIPs()
	if err != nil {
		return err
	}

	selected := pickRandom(ips, flipCount, rng)
	toDown := 0
	toUp := 0
	for _, ip := range selected {
		if downSet[ip] {
			if _, err := deleteDownIP(ip); err != nil {
				return err
			}
			delete(downSet, ip)
			toUp++
			continue
		}
		if out, err := addDownIP(ip); err != nil {
			log.Printf("nft add failed for %s: %v, output: %s", ip, err, out)
			return err
		}
		downSet[ip] = true
		toDown++
	}

	log.Printf("auto flap: flipped=%d to_down=%d to_up=%d down=%d/%d", len(selected), toDown, toUp, len(downSet), totalIPs)
	return nil
}

func flushDownIPs() ([]byte, error) {
	nftMu.Lock()
	defer nftMu.Unlock()
	cmd := exec.Command("nft", "flush", "set", "inet", "sim", "down_ips")
	return cmd.CombinedOutput()
}

func addDownIP(ip string) ([]byte, error) {
	nftMu.Lock()
	defer nftMu.Unlock()
	cmd := exec.Command("nft", "add", "element", "inet", "sim", "down_ips", "{", ip, "}")
	return cmd.CombinedOutput()
}

func deleteDownIP(ip string) ([]byte, error) {
	nftMu.Lock()
	defer nftMu.Unlock()
	cmd := exec.Command("nft", "delete", "element", "inet", "sim", "down_ips", "{", ip, "}")
	return cmd.CombinedOutput()
}

func listDownIPs() (map[string]bool, error) {
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

func cleanIPs(ips []string) []string {
	result := make([]string, 0, len(ips))
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if ip != "" {
			result = append(result, ip)
		}
	}
	return result
}

func generateIPs(count int, subnet string) []string {
	ips := make([]string, 0, count)
	octet3 := 0
	octet4 := 1
	for i := 0; i < count; i++ {
		ips = append(ips, subnet+"."+strconv.Itoa(octet3)+"."+strconv.Itoa(octet4))
		octet4++
		if octet4 > 254 {
			octet4 = 1
			octet3++
		}
	}
	return ips
}

func pickRandom(ips []string, n int, rng *rand.Rand) []string {
	if n > len(ips) {
		n = len(ips)
	}
	if n <= 0 {
		return nil
	}
	perm := rng.Perm(len(ips))
	result := make([]string, n)
	for i := 0; i < n; i++ {
		result[i] = ips[perm[i]]
	}
	return result
}

func envStr(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envBool(key string, defaultVal bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return defaultVal
}

func envFloat(key string, defaultVal float64) float64 {
	if v := os.Getenv(key); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err == nil {
			return f
		}
	}
	return defaultVal
}

func envDuration(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		if seconds, err := strconv.Atoi(v); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultVal
}
