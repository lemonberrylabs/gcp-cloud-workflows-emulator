// Package main is the entry point for the GCW emulator server.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/lemonberrylabs/gcp-cloud-workflows-emulator/pkg/api"
	grpcapi "github.com/lemonberrylabs/gcp-cloud-workflows-emulator/pkg/api/grpc"
	"github.com/lemonberrylabs/gcp-cloud-workflows-emulator/pkg/store"
	"github.com/lemonberrylabs/gcp-cloud-workflows-emulator/web"
)

func main() {
	portFlag := flag.Int("port", 0, "HTTP server port (default 8787, env PORT)")
	grpcPortFlag := flag.Int("grpc-port", 0, "gRPC server port (default 8788, env GRPC_PORT)")
	hostFlag := flag.String("host", "", "Bind address (default 0.0.0.0, env HOST)")
	projectFlag := flag.String("project", "", "GCP project ID for API paths (default my-project, env PROJECT)")
	locationFlag := flag.String("location", "", "GCP location for API paths (default us-central1, env LOCATION)")
	workflowsDirFlag := flag.String("workflows-dir", "", "Directory of workflow YAML/JSON files to watch (env WORKFLOWS_DIR)")
	flag.Parse()

	port := envOrDefault("PORT", "8787")
	if *portFlag != 0 {
		port = fmt.Sprintf("%d", *portFlag)
	}

	grpcPort := envOrDefault("GRPC_PORT", "8788")
	if *grpcPortFlag != 0 {
		grpcPort = fmt.Sprintf("%d", *grpcPortFlag)
	}

	host := envOrDefault("HOST", "0.0.0.0")
	if *hostFlag != "" {
		host = *hostFlag
	}

	project := envOrDefault("PROJECT", "my-project")
	if *projectFlag != "" {
		project = *projectFlag
	}

	location := envOrDefault("LOCATION", "us-central1")
	if *locationFlag != "" {
		location = *locationFlag
	}

	workflowsDir := os.Getenv("WORKFLOWS_DIR")
	if *workflowsDirFlag != "" {
		workflowsDir = *workflowsDirFlag
	}

	addr := fmt.Sprintf("%s:%s", host, port)
	grpcAddr := fmt.Sprintf("%s:%s", host, grpcPort)

	s := store.New()
	server := api.New(s)

	// Load workflows from directory if specified
	if workflowsDir != "" {
		log.Printf("Watching workflows directory: %s", workflowsDir)
		if err := server.WatchDir(workflowsDir, project, location); err != nil {
			log.Printf("Warning: failed to watch workflows directory: %v", err)
		}
	}

	// Register the web UI (non-fatal if template parsing fails)
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Warning: web UI disabled due to template error: %v", r)
			}
		}()
		ui := web.New(s, project, location)
		ui.Register(server.App())
	}()

	// Start gRPC server
	grpcServer := grpcapi.New(s)
	go func() {
		log.Printf("gRPC server listening on %s", grpcAddr)
		if err := grpcServer.Serve(grpcAddr); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down emulator...")
		grpcServer.GracefulStop()
		if err := server.Shutdown(); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
	}()

	log.Printf("GCW Emulator listening on %s (project=%s, location=%s)", addr, project, location)
	if workflowsDir != "" {
		log.Printf("Workflows directory: %s", workflowsDir)
	} else {
		log.Printf("API-only mode (no --workflows-dir specified)")
	}
	if err := server.Listen(addr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
