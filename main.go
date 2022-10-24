package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

var configFilePTR *string
var ngrokPathPTR *string
var wrapperConfigPTR *string
var wg sync.WaitGroup
var web_addr string = "6075"
var discordWebhookURLPTR *string

func init() {
	switch runtime.GOOS {
	case "windows":
		ngrokPath := ".\\ngrok.exe"
		configFile := ".\\ngrok.yml"
		wrapperConfig := ".\\wrapper.conf"

		ngrokPathPTR = &ngrokPath
		configFilePTR = &configFile
		wrapperConfigPTR = &wrapperConfig
	case "linux":
		ngrokPath := "./ngrok"
		configFile := "./ngrok.yml"
		wrapperConfig := "./wrapper.conf"

		ngrokPathPTR = &ngrokPath
		configFilePTR = &configFile
		wrapperConfigPTR = &wrapperConfig
	default:
		fmt.Println("Your OS is not supported.")
		os.Exit(66)
	}

	contentBytes, err := os.ReadFile(*wrapperConfigPTR)
	if err != nil {
		configureDiscordWebhook()
		os.Exit(0)
	}
	if ! strings.Contains(string(contentBytes), "https://discordapp.com/api/webhooks") && ! strings.Contains(string(contentBytes), "https://discord.com/api/webhooks") {
		configureDiscordWebhook()
		os.Exit(0)
	}
	discordWebhookURL := strings.ReplaceAll(string(contentBytes), "\n", "")
	discordWebhookURLPTR = &discordWebhookURL
}

func configureDiscordWebhook() {
	scanner := bufio.NewScanner(os.Stdin)
	
	fmt.Printf("Please input discord webhook url: ")
	scanner.Scan()
	err := os.WriteFile(*wrapperConfigPTR, []byte(scanner.Text()), 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	switch os.Args[1] {
	case "--configure":
		createConfig()
	case "--help":
		printHelp()
		return
	case "http":
		if ! checkconfigFilePTR() {
			fmt.Println("Please add your authtoken by using the --configure command")
			return
		}
		commandArgs := append(os.Args[1:], "--config", *configFilePTR)

		cmd := exec.Command(*ngrokPathPTR, commandArgs...)
		go checkOutput(cmd, &wg)
		wg.Add(1)

		getNgrokAddress()

		wg.Wait()
		return
	case "tcp":
		if ! checkconfigFilePTR() {
			fmt.Println("Please add your authtoken by using the --configure command")
			return
		}
		commandArgs := append(os.Args[1:], "--config", *configFilePTR)

		cmd := exec.Command(*ngrokPathPTR, commandArgs...)
		go checkOutput(cmd, &wg)
		wg.Add(1)

		getNgrokAddress()

		wg.Wait()
		return
	default:
		fmt.Printf("Could not find command %v, do you want to execute it using ngrok? [y/N]: ", os.Args[1])
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		if scanner.Text() != "y" && scanner.Text() != "Y" {
			printHelp()
			return
		}
		fmt.Println("Not implemented yet...")
		return
	}
}

type DiscordMessage struct {
	Username string `json:"username"`
	Content string `json:"content"`
}

func sendWebHook(payload DiscordMessage) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", *discordWebhookURLPTR, body)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
}

type NgrokAPI struct {
	Tunnels []struct {
		Name      string `json:"name"`
		ID        string `json:"ID"`
		URI       string `json:"uri"`
		PublicURL string `json:"public_url"`
		Proto     string `json:"proto"`
		Config    struct {
			Addr    string `json:"addr"`
			Inspect bool   `json:"inspect"`
		} `json:"config"`
		Metrics struct {
			Conns struct {
				Count  int `json:"count"`
				Gauge  int `json:"gauge"`
				Rate1  int `json:"rate1"`
				Rate5  int `json:"rate5"`
				Rate15 int `json:"rate15"`
				P50    int `json:"p50"`
				P90    int `json:"p90"`
				P95    int `json:"p95"`
				P99    int `json:"p99"`
			} `json:"conns"`
			HTTP struct {
				Count  int `json:"count"`
				Rate1  int `json:"rate1"`
				Rate5  int `json:"rate5"`
				Rate15 int `json:"rate15"`
				P50    int `json:"p50"`
				P90    int `json:"p90"`
				P95    int `json:"p95"`
				P99    int `json:"p99"`
			} `json:"http"`
		} `json:"metrics"`
	} `json:"tunnels"`
	URI string `json:"uri"`
}

func getNgrokAddress() {
	time.Sleep(3 * time.Second)

	fmt.Println("Checking address...")
	url := "http://127.0.0.1:" + web_addr + "/api/tunnels"

	client := http.Client{Timeout: time.Second * 3}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		panic(err)
	}

	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	result := &NgrokAPI{}
    err = json.Unmarshal(body, result)
    if err != nil {
        panic(err)
    }

    payload := DiscordMessage{
    	Username: "Ngrok",
    	Content: result.Tunnels[0].PublicURL,
    }

    sendWebHook(payload)
}

func checkOutput(cmd *exec.Cmd, wg *sync.WaitGroup) {
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error:", string(out))
		wg.Done()
	}
}

func printHelp() {
	fmt.Println("EXAMPLES:")
	fmt.Println("   ngrok-wrapper --configure        # Add authtoken to use with ngrok")
	fmt.Println("   ngrok-wrapper http 80            # Secure public URL for port 80")
	fmt.Println("   ngrok-wrapper tcp 22             # Tunnel arbitrary TCP traffic to port 22")
}

func createConfig() {
	if checkconfigFilePTR() {
		fmt.Println("There already is a config file at", *configFilePTR)
		fmt.Printf("If you continue, you will overwrite the existing config.\n\n")
	}
	scanner := bufio.NewScanner(os.Stdin)
	
	fmt.Printf("Please input authtoken: ")
	scanner.Scan()
	authtoken := scanner.Text()
	version := "\"2\""

	config := []byte("version: " + version + "\nauthtoken: " + authtoken + "\nweb_addr: " + web_addr)

	fmt.Println("Saving to file...")
	err := os.WriteFile(*configFilePTR, config, 0644)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Saved config to %v using authtoken %v, version %v and web interface port %v\n", *configFilePTR, authtoken, version, web_addr)
}

func checkconfigFilePTR() (exists bool) {
	if _, err := os.Stat(*configFilePTR); os.IsNotExist(err) {
		return false
	}
	return true
}