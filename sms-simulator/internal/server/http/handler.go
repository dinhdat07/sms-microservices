package http

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"sms-simulator/internal/nft"
)

type Server struct {
	TotalIPs int
	Subnet   string
}

type toggleRequest struct {
	IPs []string `json:"ips"`
}

type statusResponse struct {
	Total int `json:"total"`
	Down  int `json:"down"`
}

func NewServer(totalIPs int, subnet string) *Server {
	return &Server{
		TotalIPs: totalIPs,
		Subnet:   subnet,
	}
}

func (s *Server) SetupMux() *http.ServeMux {
	mux := http.ServeMux{}
	mux.HandleFunc("/up", s.handleUp)
	mux.HandleFunc("/down", s.handleDown)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/reset", s.handleReset)

	// Add /health endpoint for AGENT_PULL
	mux.HandleFunc("/health", s.handleHealth)

	return &mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// If the request reached here, nftables didn't drop it. So it's healthy.
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status": "ok"}`))
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if out, err := nft.FlushDownIPs(); err != nil {
		log.Printf("nft flush failed: %v, output: %s", err, out)
		http.Error(w, "nft error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleDown(w http.ResponseWriter, r *http.Request) {
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
		if out, err := nft.AddDownIP(ip); err != nil {
			log.Printf("nft add failed for %s: %v, output: %s", ip, err, out)
			http.Error(w, "nft error", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleUp(w http.ResponseWriter, r *http.Request) {
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
		_, _ = nft.DeleteDownIP(ip)
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	downSet, err := nft.ListDownIPs()
	if err != nil {
		log.Printf("nft list failed: %v", err)
		http.Error(w, "nft error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(statusResponse{
		Total: s.TotalIPs,
		Down:  len(downSet),
	})
}

func (s *Server) StartAutoFlapper() {
	interval := envDuration("SIMULATOR_FLAP_INTERVAL", 60*time.Second)
	percent := envFloat("SIMULATOR_FLAP_PERCENT", 5)
	if interval <= 0 || percent <= 0 {
		log.Printf("Auto flapper disabled: interval=%s percent=%.2f", interval, percent)
		return
	}

	ips := generateIPs(s.TotalIPs, s.Subnet)
	flipCount := int(float64(s.TotalIPs) * percent / 100)
	if flipCount < 1 && s.TotalIPs > 0 {
		flipCount = 1
	}
	if flipCount > s.TotalIPs {
		flipCount = s.TotalIPs
	}

	seed := time.Now().UnixNano()
	if v := os.Getenv("SIMULATOR_FLAP_SEED"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			seed = n
		}
	}
	rng := rand.New(rand.NewSource(seed))

	log.Printf("Auto flapper enabled: interval=%s, flip=%.2f%% (%d/%d), seed=%d", interval, percent, flipCount, s.TotalIPs, seed)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if err := flapOnce(ips, flipCount, rng, s.TotalIPs); err != nil {
				log.Printf("auto flap failed: %v", err)
			}
		}
	}()
}

func flapOnce(ips []string, flipCount int, rng *rand.Rand, totalIPs int) error {
	downSet, err := nft.ListDownIPs()
	if err != nil {
		return err
	}

	selected := pickRandom(ips, flipCount, rng)
	toDown := 0
	toUp := 0
	for _, ip := range selected {
		if downSet[ip] {
			if _, err := nft.DeleteDownIP(ip); err != nil {
				return err
			}
			delete(downSet, ip)
			toUp++
			continue
		}
		if out, err := nft.AddDownIP(ip); err != nil {
			log.Printf("nft add failed for %s: %v, output: %s", ip, err, out)
			return err
		}
		downSet[ip] = true
		toDown++
	}

	log.Printf("auto flap: flipped=%d to_down=%d to_up=%d down=%d/%d", len(selected), toDown, toUp, len(downSet), totalIPs)
	return nil
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
