package command

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/langgenius/dify-cli/config"
	"github.com/langgenius/dify-cli/tool"
	"github.com/langgenius/dify-cli/types"
)

func Execute() {
	args := os.Args[1:]

	invokedName := filepath.Base(os.Args[0])
	ref, isSymlink := tryDetectSymlink(invokedName)

	if isSymlink && ref != nil {
		handleToolInvocation(ref)
		return
	}

	if len(args) == 0 {
		printMainHelp("")
		return
	}

	subcommand := args[0]

	if subcommand == "--help" || subcommand == "-h" || subcommand == "help" {
		if len(args) > 1 {
			runHelp(args[1])
		} else {
			printMainHelp("")
		}
		return
	}

	switch subcommand {
	case "init":
		runInit()
	case "list":
		runList()
	case "env":
		runEnv()
	case "execute":
		if len(args) > 1 {
			runExecute(args[1])
		} else {
			printMainHelp("missing tool name")
		}
	default:
		printMainHelp(fmt.Sprintf("unknown command: %s", subcommand))
	}
}

func tryDetectSymlink(name string) (*types.ToolReference, bool) {
	if ref, err := config.FindToolReference(name); err == nil {
		return ref, true
	}
	return nil, false
}

func handleToolInvocation(ref *types.ToolReference) {
	args := os.Args[1:]

	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			cfg, err := config.Load()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			info, err := tool.FetchToolInfo(ref, &cfg.Env)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			tool.PrintHelp(info)
			return
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	params, err := tool.ParseArgs(ref, args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := tool.Dispatch(ref, &cfg.Env, params); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runInit() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.ToolReferences) == 0 {
		fmt.Println("No tool references defined in config")
		return
	}

	selfPath, err := config.GetSelfPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get self path: %v\n", err)
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	created := 0
	skipped := 0

	for _, ref := range cfg.ToolReferences {
		symlinkName := config.GetReferenceSymlinkName(ref)
		symlinkPath := filepath.Join(cwd, symlinkName)

		if _, err := os.Lstat(symlinkPath); err == nil {
			skipped++
			continue
		}

		if err := os.Symlink(selfPath, symlinkPath); err != nil {
			fmt.Fprintf(os.Stderr, "  [FAIL] %s: %v\n", symlinkName, err)
			continue
		}

		created++
	}

	fmt.Printf("Created %d symlinks, skipped %d\n", created, skipped)
	if created > 0 {
		fmt.Println("After initialization, you can use symlinked commands directly:")
		fmt.Println("  execute <tool_name> [--param value ...]")
	}
}

func runList() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.ToolReferences) == 0 {
		fmt.Println("No tools configured")
		return
	}

	fmt.Println("Available tools:")
	for _, ref := range cfg.ToolReferences {
		symlinkName := config.GetReferenceSymlinkName(ref)
		fmt.Printf("  %s (type: %s, provider: %s)\n",
			symlinkName, ref.ToolType, ref.ToolProvider)
	}
}

func runEnv() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	configPath, _ := config.GetConfigPath()
	fmt.Printf("Config File:      %s\n", configPath)
	fmt.Printf("Files URL:        %s\n", cfg.Env.FilesURL)
	fmt.Printf("CLI API URL:      %s\n", cfg.Env.CliApiURL)
	fmt.Printf("Session ID:       %s\n", cfg.Env.CliApiSessionID)
	fmt.Printf("Tool References:  %d\n", len(cfg.ToolReferences))
}

func runHelp(toolName string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	for _, ref := range cfg.ToolReferences {
		newRef := ref
		if ref.ToolName == toolName || config.GetReferenceSymlinkName(ref) == toolName {
			info, err := tool.FetchToolInfo(&newRef, &cfg.Env)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			tool.PrintHelp(info)
			return
		}
	}

	fmt.Fprintf(os.Stderr, "Error: tool not found: %s\n", toolName)
	os.Exit(1)
}

func runExecute(toolName string) {
	if ref, err := config.FindToolReference(toolName); err == nil {
		handleToolInvocation(ref)
		return
	}

	fmt.Fprintf(os.Stderr, "Error: tool reference not found: %s (must use format: tool_name_uuid)\n", toolName)
	os.Exit(1)
}

func printMainHelp(errMsg string) {
	if errMsg != "" {
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", errMsg)
	}

	fmt.Println(`Dify CLI for sandbox tool invocation

Usage:
  dify [command]

Commands:
  init        Initialize tool symlinks from config
  list        List all available tool references from the config
  env         Show current environment configuration
  help <tool> Show help for a specific tool
  execute <tool_ref> [--param value ...]   Execute a tool directly

Environment:
  DIFY_CLI_CONFIG    Path to .dify_cli.json (default: CWD/.dify_cli.json)`)
}
