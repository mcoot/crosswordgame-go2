package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

func newEventsCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "events <code>",
		Short: "Stream SSE events from a lobby",
		Long: `Connect to the lobby's SSE endpoint and stream events in real-time.

Events include:
  - member-update: Lobby member list changed
  - game-started: Game has started
  - letter-announced: New letter announced
  - placement-update: Player placed letter
  - turn-complete: All players placed, new turn
  - game-complete: Game finished
  - game-abandoned: Game was abandoned
  - refresh: Generic refresh signal

Press Ctrl+C to disconnect.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code := args[0]
			return streamEvents(code, jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output events as JSON lines")

	return cmd
}

// SSEEvent represents a parsed SSE event
type SSEEvent struct {
	Time  time.Time `json:"time"`
	Event string    `json:"event"`
	Data  string    `json:"data"`
}

func streamEvents(lobbyCode string, jsonOutput bool) error {
	// Build SSE URL - note: SSE is on the web router, not the API router
	url := strings.TrimSuffix(cfg.ServerURL, "/") + "/lobby/" + lobbyCode + "/events"

	// Create request
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	// Add session cookie for web auth
	if cfg.Token != "" {
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: cfg.Token,
		})
	}

	// Set up cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	req = req.WithContext(ctx)

	// Make request
	httpClient := &http.Client{
		Timeout: 0, // No timeout for SSE
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	if !jsonOutput {
		fmt.Printf("Connected to lobby %s\n", lobbyCode)
	}

	// Parse SSE stream
	scanner := bufio.NewScanner(resp.Body)
	var currentEvent string
	var dataLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			dataLines = append(dataLines, strings.TrimPrefix(line, "data: "))
		} else if line == "" {
			// End of event
			if currentEvent != "" {
				data := strings.Join(dataLines, "\n")
				printEvent(currentEvent, data, jsonOutput)
			}
			currentEvent = ""
			dataLines = nil
		}
	}

	if err := scanner.Err(); err != nil {
		// Context cancellation is expected
		if ctx.Err() != nil {
			if !jsonOutput {
				fmt.Println("\nDisconnected")
			}
			return nil
		}
		return fmt.Errorf("stream error: %w", err)
	}

	if !jsonOutput {
		fmt.Println("Disconnected")
	}
	return nil
}

func printEvent(event, data string, jsonOutput bool) {
	now := time.Now()

	if jsonOutput {
		evt := SSEEvent{
			Time:  now,
			Event: event,
			Data:  data,
		}
		jsonData, _ := json.Marshal(evt)
		fmt.Println(string(jsonData))
	} else {
		timestamp := now.Format("2006-01-02 15:04:05")
		// Truncate data if it's too long for display
		displayData := data
		if len(displayData) > 100 {
			displayData = displayData[:100] + "..."
		}
		// Remove newlines for cleaner display
		displayData = strings.ReplaceAll(displayData, "\n", " ")
		fmt.Printf("[%s] %s: %s\n", timestamp, event, displayData)
	}
}
