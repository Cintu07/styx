package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const defaultServer = "http://localhost:8080"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	server := os.Getenv("STYX_SERVER")
	if server == "" {
		server = defaultServer
	}

	cmd := os.Args[1]

	switch cmd {
	case "query":
		if len(os.Args) < 3 {
			fmt.Println("usage: styx query <node_id>")
			os.Exit(1)
		}
		query(server, os.Args[2])

	case "report":
		if len(os.Args) < 6 {
			fmt.Println("usage: styx report <witness> <target> <alive> <dead> <unknown>")
			os.Exit(1)
		}
		report(server, os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6])

	case "health":
		health(server)

	case "version":
		fmt.Println("styx cli v1.0.0")

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("styx - truthful membership cli")
	fmt.Println("")
	fmt.Println("usage:")
	fmt.Println("  styx query <node_id>                           query node status")
	fmt.Println("  styx report <witness> <target> <a> <d> <u>     submit witness report")
	fmt.Println("  styx health                                    check server health")
	fmt.Println("  styx version                                   show version")
	fmt.Println("")
	fmt.Println("environment:")
	fmt.Println("  STYX_SERVER    server url (default: http://localhost:8080)")
}

func query(server, nodeID string) {
	resp, err := http.Get(server + "/query?target=" + nodeID)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	alive := result["alive_confidence"].(float64)
	dead := result["dead_confidence"].(float64)
	unknown := result["unknown"].(float64)
	refused := result["refused"].(bool)

	fmt.Printf("node %s:\n", nodeID)
	fmt.Printf("  alive:   %.1f%%\n", alive*100)
	fmt.Printf("  dead:    %.1f%%\n", dead*100)
	fmt.Printf("  unknown: %.1f%%\n", unknown*100)

	if refused {
		fmt.Printf("  status:  REFUSED (%s)\n", result["refusal_reason"])
	} else if result["dead"].(bool) {
		fmt.Printf("  status:  DEAD (finality)\n")
	} else if alive > dead && alive > unknown {
		fmt.Printf("  status:  likely alive\n")
	} else if dead > alive && dead > unknown {
		fmt.Printf("  status:  likely dead\n")
	} else {
		fmt.Printf("  status:  uncertain\n")
	}
}

func report(server, witness, target, alive, dead, unknown string) {
	a, _ := strconv.ParseFloat(alive, 64)
	d, _ := strconv.ParseFloat(dead, 64)
	u, _ := strconv.ParseFloat(unknown, 64)

	body := fmt.Sprintf(`{"witness":%s,"target":%s,"alive":%f,"dead":%f,"unknown":%f}`,
		witness, target, a, d, u)

	resp, err := http.Post(server+"/report", "application/json", strings.NewReader(body))
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 202 {
		fmt.Println("report accepted")
	} else {
		fmt.Printf("error: status %d\n", resp.StatusCode)
	}
}

func health(server string) {
	resp, err := http.Get(server + "/health")
	if err != nil {
		fmt.Printf("error: server unreachable\n")
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		fmt.Println("server: healthy")
	} else {
		fmt.Printf("server: unhealthy (status %d)\n", resp.StatusCode)
	}
}
