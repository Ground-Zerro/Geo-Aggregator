package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	speedtestMapPath  = "db/speedtest.map.yml"
	speedtestPath     = "db/speedtest.json"
	speedtestVersion  = 1
	libreSpeedServers = "https://librespeed.org/backend-servers/servers.php"
	aliveTimeout      = 8 * time.Second
	aliveParallel     = 16
	aliveProbeBytes   = 64 << 10
)

var libreSpeedCountries = map[string]string{
	"Australia": "AU", "Austria": "AT", "Belgium": "BE", "Brazil": "BR", "Bulgaria": "BG", "Canada": "CA",
	"Czech Republic": "CZ", "Czechia": "CZ", "Denmark": "DK", "England": "GB", "Estonia": "EE", "Finland": "FI",
	"France": "FR", "Germany": "DE", "Greece": "GR", "Hong Kong": "HK", "Hungary": "HU", "India": "IN",
	"Ireland": "IE", "Italy": "IT", "Japan": "JP", "Kazakhstan": "KZ", "Latvia": "LV", "Lithuania": "LT",
	"Netherlands": "NL", "Norway": "NO", "Poland": "PL", "Portugal": "PT", "Romania": "RO", "Russia": "RU",
	"Serbia": "RS", "Singapore": "SG", "Spain": "ES", "Sweden": "SE", "Switzerland": "CH", "Turkey": "TR",
	"UK": "GB", "Ukraine": "UA", "United Kingdom": "GB", "United States": "US", "USA": "US",
}

type speedIperf3 struct {
	Host  string `yaml:"host" json:"host"`
	Ports string `yaml:"ports" json:"ports"`
}

type speedLibreSpeed struct {
	Download string `yaml:"dl" json:"dl"`
	Upload   string `yaml:"ul" json:"ul"`
	Ping     string `yaml:"ping" json:"ping"`
}

type speedNode struct {
	Country    string           `yaml:"country" json:"country"`
	City       string           `yaml:"city" json:"city"`
	Owner      string           `yaml:"owner" json:"owner"`
	Iperf3     *speedIperf3     `yaml:"iperf3" json:"iperf3,omitempty"`
	LibreSpeed *speedLibreSpeed `yaml:"librespeed" json:"librespeed,omitempty"`
	File       string           `yaml:"file" json:"file,omitempty"`
}

type speedtestFile struct {
	Version int         `json:"version"`
	Updated string      `json:"updated"`
	Nodes   []speedNode `json:"nodes"`
}

func buildSpeedtest(now time.Time) int {
	data, err := os.ReadFile(speedtestMapPath)
	if err != nil {
		fmt.Println(" ", err)
		return 0
	}
	var nodes []speedNode
	if err := yaml.Unmarshal(data, &nodes); err != nil {
		fmt.Println(" ", speedtestMapPath+":", err)
		return 0
	}
	nodes = append(nodes, importLibreSpeed()...)
	alive := checkSpeedNodes(nodes)
	sort.SliceStable(alive, func(i, j int) bool {
		a, b := alive[i], alive[j]
		if a.Country != b.Country {
			return a.Country < b.Country
		}
		if a.City != b.City {
			return a.City < b.City
		}
		return a.Owner < b.Owner
	})
	out, err := json.MarshalIndent(speedtestFile{Version: speedtestVersion, Updated: now.Format(time.RFC3339), Nodes: alive}, "", "  ")
	if err != nil {
		fmt.Println(" ", err)
		return 0
	}
	if err := os.WriteFile(speedtestPath, append(out, '\n'), 0644); err != nil {
		fmt.Println(" ", err)
		return 0
	}
	fmt.Printf("  %d узлов из %d\n", len(alive), len(nodes))
	return len(alive)
}

func importLibreSpeed() []speedNode {
	data, err := fetch(libreSpeedServers)
	if err != nil {
		fmt.Println("  LibreSpeed:", err)
		return nil
	}
	var servers []struct {
		Name    string `json:"name"`
		Server  string `json:"server"`
		DlURL   string `json:"dlURL"`
		UlURL   string `json:"ulURL"`
		PingURL string `json:"pingURL"`
		Sponsor string `json:"sponsorName"`
	}
	if err := json.Unmarshal(data, &servers); err != nil {
		fmt.Println("  LibreSpeed:", err)
		return nil
	}
	var nodes []speedNode
	for _, s := range servers {
		parts := strings.Split(stripParens(s.Name), ",")
		country, ok := libreSpeedCountries[strings.TrimSpace(parts[len(parts)-1])]
		if !ok || len(parts) < 2 {
			fmt.Println("  LibreSpeed: страна не распознана:", s.Name)
			continue
		}
		base := strings.TrimSuffix(s.Server, "/") + "/"
		nodes = append(nodes, speedNode{
			Country: country,
			City:    strings.TrimSpace(parts[0]),
			Owner:   s.Sponsor,
			LibreSpeed: &speedLibreSpeed{
				Download: base + strings.TrimPrefix(s.DlURL, "/"),
				Upload:   base + strings.TrimPrefix(s.UlURL, "/"),
				Ping:     base + strings.TrimPrefix(s.PingURL, "/"),
			},
		})
	}
	return nodes
}

func stripParens(s string) string {
	for {
		open := strings.Index(s, "(")
		if open < 0 {
			return s
		}
		end := strings.Index(s[open:], ")")
		if end < 0 {
			return s[:open]
		}
		s = s[:open] + s[open+end+1:]
	}
}

func checkSpeedNodes(nodes []speedNode) []speedNode {
	alive := make([]*speedNode, len(nodes))
	slots := make(chan struct{}, aliveParallel)
	var wg sync.WaitGroup
	for i := range nodes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			n := nodes[i]
			if n.Iperf3 != nil && !iperf3Alive(n.Iperf3) {
				fmt.Println("  нет ответа iperf3:", n.Owner, n.City, n.Iperf3.Host)
				n.Iperf3 = nil
			}
			if n.LibreSpeed != nil && !httpAlive(n.LibreSpeed.Download+"?ckSize=1") {
				fmt.Println("  нет ответа LibreSpeed:", n.Owner, n.City, n.LibreSpeed.Download)
				n.LibreSpeed = nil
			}
			if n.File != "" && !httpAlive(n.File) {
				fmt.Println("  нет ответа файла:", n.Owner, n.City, n.File)
				n.File = ""
			}
			if n.Iperf3 != nil || n.LibreSpeed != nil || n.File != "" {
				alive[i] = &n
			}
		}(i)
	}
	wg.Wait()
	var out []speedNode
	for _, n := range alive {
		if n != nil {
			out = append(out, *n)
		}
	}
	return out
}

func iperf3Alive(p *speedIperf3) bool {
	lo, hi, ok := strings.Cut(p.Ports, "-")
	if !ok {
		hi = lo
	}
	from, errLo := strconv.Atoi(lo)
	to, errHi := strconv.Atoi(hi)
	if errLo != nil || errHi != nil || to < from {
		return false
	}
	for port := from; port <= to && port < from+3; port++ {
		conn, err := net.DialTimeout("tcp4", net.JoinHostPort(p.Host, strconv.Itoa(port)), aliveTimeout)
		if err == nil {
			conn.Close()
			return true
		}
	}
	return false
}

var aliveClient = &http.Client{Transport: &http.Transport{
	DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
		return (&net.Dialer{Timeout: aliveTimeout}).DialContext(ctx, "tcp4", addr)
	},
	TLSHandshakeTimeout: aliveTimeout,
}}

func httpAlive(url string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), aliveTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := aliveClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	n, _ := io.CopyN(io.Discard, resp.Body, aliveProbeBytes)
	return resp.StatusCode == http.StatusOK && n == aliveProbeBytes
}

func runSpeedtestOnly() {
	fmt.Println("=== Генерация db/speedtest.json ===")
	buildSpeedtest(time.Now().UTC())
}
