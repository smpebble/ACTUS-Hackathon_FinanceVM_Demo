// Command server starts the ACTUS-FVM web demo server.
// Run from the project root: go run cmd/server/main.go
// Then open http://localhost:8080 in your browser.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/smpebble/actus-fvm/internal/api"
)

func main() {
	// Use the current working directory as the project root.
	// Always run this command from the ACTUS-FVM project root.
	baseDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("cannot determine working directory: %v", err)
	}
	if dir := os.Getenv("ACTUS_BASE_DIR"); dir != "" {
		baseDir = dir
	}

	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}

	// Auto-install npm dependencies for the Solidity compiler if missing.
	scriptsDir := filepath.Join(baseDir, "scripts")
	if _, err := os.Stat(filepath.Join(scriptsDir, "node_modules")); os.IsNotExist(err) {
		fmt.Println("  Installing npm dependencies for Solidity compiler...")
		npmCmd := exec.Command("npm", "install")
		npmCmd.Dir = scriptsDir
		if out, err := npmCmd.CombinedOutput(); err != nil {
			fmt.Printf("  Warning: npm install failed: %v\n%s\n", err, out)
		} else {
			fmt.Println("  npm install complete.")
		}
	}

	server := api.NewServer(baseDir)

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║       ACTUS-FVM  Web Demo  —  ACTUS Hackathon 2025          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Printf("  Listening on  http://localhost%s\n", addr)
	fmt.Printf("  Base dir      %s\n\n", baseDir)
	fmt.Println("  Open the URL above in your browser to run the demo.")
	fmt.Println("  Press Ctrl+C to stop.")
	fmt.Println()

	if err := http.ListenAndServe(addr, server); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
