// Command voban configures AI coding tools to use the Voban gateway.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/vobanai/voban-cli/internal/client"
	"github.com/vobanai/voban-cli/internal/config"
	"github.com/vobanai/voban-cli/internal/opencode"
)

const usage = `voban configures AI coding tools to use the Voban gateway.

Usage:
  voban configure opencode   Configure opencode to use Voban (writes opencode config)
  voban models               List the models available to your API key
  voban status               Show your identity, budget, and spend

The API key is read from the VOBAN_API_KEY environment variable, from a prior
configure run, or prompted interactively. Set VOBAN_BASE_URL to target a
self-hosted gateway (default: https://api.voban.ai).
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "voban: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	switch args[0] {
	case "configure":
		return runConfigure(ctx, args[1:])
	case "models":
		return runModels(ctx)
	case "status":
		return runStatus(ctx)
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	default:
		return fmt.Errorf("unknown command %q (run 'voban help')", args[0])
	}
}

func runConfigure(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("configure requires a tool name (supported: opencode)")
	}
	if args[0] != "opencode" {
		return fmt.Errorf("unsupported tool %q (supported: opencode)", args[0])
	}

	key, err := resolveKey(false)
	if err != nil {
		return err
	}

	models, err := opencode.Configure(ctx, key)
	if err != nil {
		return err
	}

	fmt.Printf("Configured opencode for Voban with %d model(s):\n", len(models))
	for _, m := range models {
		fmt.Printf("  - %s\n", m)
	}
	fmt.Println("\nStart opencode and run /models to pick a Voban model.")
	return nil
}

func runModels(ctx context.Context) error {
	key, err := resolveKey(true)
	if err != nil {
		return err
	}
	models, err := client.New(config.BaseURL(), key).Models(ctx)
	if err != nil {
		return err
	}
	if len(models) == 0 {
		fmt.Println("No models available for this key.")
		return nil
	}
	for _, m := range models {
		fmt.Println(m)
	}
	return nil
}

func runStatus(ctx context.Context) error {
	key, err := resolveKey(true)
	if err != nil {
		return err
	}
	c := client.New(config.BaseURL(), key)

	me, err := c.Me(ctx)
	if err != nil {
		return fmt.Errorf("get identity: %w", err)
	}
	fmt.Printf("User:     %s\n", me.UserID)
	if me.Email != "" {
		fmt.Printf("Email:    %s\n", me.Email)
	}
	fmt.Printf("Customer: %t\n", me.CustomerExists)

	spend, err := c.Spend(ctx)
	if err != nil {
		return fmt.Errorf("get spend: %w", err)
	}
	fmt.Printf("Spend:    %.4f\n", spend.Spend)
	if spend.MaxBudget > 0 {
		fmt.Printf("Budget:   %.4f", spend.MaxBudget)
		if spend.BudgetDuration != "" {
			fmt.Printf(" per %s", spend.BudgetDuration)
		}
		fmt.Println()
	}
	fmt.Printf("Blocked:  %t\n", spend.Blocked)
	return nil
}

// resolveKey finds the API key from VOBAN_API_KEY, then opencode's stored auth
// (when reuseStored is set), then an interactive prompt. The key is validated
// before it is returned.
func resolveKey(reuseStored bool) (string, error) {
	if key := strings.TrimSpace(os.Getenv("VOBAN_API_KEY")); key != "" {
		return key, config.ValidateAPIKey(key)
	}
	if reuseStored {
		key, ok, err := opencode.StoredKey()
		if err != nil {
			return "", fmt.Errorf("read stored opencode key: %w", err)
		}
		if ok {
			return key, config.ValidateAPIKey(key)
		}
	}
	key, err := promptKey()
	if err != nil {
		return "", err
	}
	return key, config.ValidateAPIKey(key)
}

func promptKey() (string, error) {
	fmt.Fprint(os.Stderr, "Enter your Voban API key (sk-sov-...): ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("read API key: %w", err)
	}
	key := strings.TrimSpace(line)
	if key == "" {
		return "", fmt.Errorf("no API key entered")
	}
	return key, nil
}
