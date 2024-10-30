package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"openvpn-proxy/daemon"
)

func globalInit(dataDir string) {
	markerFile := "/.initialized"
	appScript := filepath.Join(dataDir, "apps.sh")

	if _, err := os.Stat(markerFile); os.IsNotExist(err) {
		if _, err := os.Stat(appScript); err == nil {
			log.Printf("Running one-time setup for apps script %s", appScript)
			cmd := exec.Command(appScript, "setup")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				log.Println(err)
			}
		} else {
			log.Println("No apps script found")
		}
		os.Create(markerFile)
	}
}

func main() {
	// Set up termination signal handler
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("SIGTERM received, shutting down...")
		daemon.ShutdownOpenVPN()
		os.Exit(0)
	}()

	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	err = os.Chdir(filepath.Dir(ex))
	if err != nil {
		log.Fatal(err)
	}

	// Command-line flag for port
	port := flag.String("port", "80", "Port to run the server on")
	dataDir := flag.String("data", "/data", "Directory to store data")
	daemon.StaticDir = flag.String("static", "./static", "Directory of static files")
	flag.Parse()

	args := flag.Args()
	if len(args) > 0 {
		// if args 0 starts with tun
		if args[0][:3] == "tun" {
			daemon.VpnUpDown()
			os.Exit(0)
		}
	}

	globalInit(*dataDir)

	daemon.Init(*dataDir)
	daemon.StartOpenVPNLoop()
	daemon.WebServer(*port)
}
