package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/branchkit/plugin-sdk-go"
)

func main() {
	logger := log.New(os.Stderr, "[basetypes] ", log.LstdFlags)

	plugin := shared.NewPlugin()

	// Open a listen port for the basetypes service to push data to.
	listener, err := shared.ListenLocal(plugin)
	if err != nil {
		logger.Fatalf("listen: %v", err)
	}
	logger.Printf("listening on %s", listener.Addr())

	// Basetypes pushes selection-mode HUD items here.
	// The path matches what basetypes already POSTs to.
	listener.HandleFunc("POST /v1/plugins/selection", func(w http.ResponseWriter, r *http.Request) {
		var payload selectionPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Forward to actuator via RPC
		if err := plugin.Call("selection.set", payload, nil); err != nil {
			logger.Printf("selection.set failed: %v", err)
			http.Error(w, "actuator unavailable", http.StatusBadGateway)
			return
		}

		shared.WriteJSON(w, map[string]bool{"ok": true})
	})

	// Start listener in background
	go func() {
		if err := listener.Serve(); err != nil {
			logger.Printf("listener error: %v", err)
		}
	}()

	logger.Printf("basetypes plugin ready")

	// Run JSON-RPC message loop (blocks until stdin closes or SIGTERM)
	plugin.Run()

	// Cleanup
	listener.Shutdown(context.Background())
	logger.Printf("basetypes plugin stopped")
}

// selectionPayload matches what basetypes pushes — title + items for HUD.
type selectionPayload struct {
	Title string    `json:"title"`
	Items []hudItem `json:"items"`
}

type hudItem struct {
	ID       string `json:"id"`
	Tag      string `json:"tag,omitempty"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	Icon     string `json:"icon,omitempty"`
}
