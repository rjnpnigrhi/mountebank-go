package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mountebank-testing/mountebank-go/internal/config"
	"github.com/mountebank-testing/mountebank-go/internal/server"
	"github.com/spf13/cobra"
)

var (
	port           int
	host           string
	logLevel       string
	allowInjection bool
	pidFile        string
	configFile     string
	saveFile       string
	logFile        string
	noLogFile      bool
	datadir        string
	impostersRepo  string
	ipWhitelist    string
	origin         []string
	apiKey         string
	debug          bool
	localOnly      bool
	protoFile      string
	rcFile         string
	formatter      string
	noParse        bool
	logConfig      string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "mb",
		Short: "mountebank - over the wire test doubles",
		Long:  `mountebank is a service virtualization tool that provides test doubles over the wire.`,
	}

	// Start command
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the mountebank server",
		Run:   runStart,
	}

	startCmd.Flags().IntVar(&port, "port", 2525, "Port to run the server on")
	startCmd.Flags().StringVar(&host, "host", "localhost", "Host to bind to")
	startCmd.Flags().StringVar(&logLevel, "loglevel", "info", "Log level (debug, info, warn, error)")
	startCmd.Flags().BoolVar(&allowInjection, "allowInjection", false, "Allow JavaScript injection")
	startCmd.Flags().StringVar(&pidFile, "pidfile", "mb.pid", "PID file location")
	startCmd.Flags().StringVar(&configFile, "configfile", "", "Configuration file to load")
	startCmd.Flags().StringVar(&logFile, "logfile", "mb.log", "Log file location")
	startCmd.Flags().BoolVar(&noLogFile, "nologfile", false, "Prevent logging to the filesystem")
	startCmd.Flags().StringVar(&datadir, "datadir", "", "The directory to save imposters to")
	startCmd.Flags().StringVar(&ipWhitelist, "ipWhitelist", "*", "IP whitelist (pipe-delimited)")
	startCmd.Flags().StringSliceVar(&origin, "origin", []string{}, "Allowed CORS origins")
	startCmd.Flags().StringVar(&apiKey, "apikey", "", "API key for authentication")
	startCmd.Flags().BoolVar(&debug, "debug", false, "Enable debug mode")
	startCmd.Flags().BoolVar(&localOnly, "localOnly", false, "Only allow connections from localhost")
	startCmd.Flags().StringVar(&protoFile, "protofile", "protocols.json", "Custom protocol file")
	startCmd.Flags().StringVar(&rcFile, "rcfile", "", "Run commands file")
	startCmd.Flags().StringVar(&formatter, "formatter", "", "Custom formatter")
	startCmd.Flags().BoolVar(&noParse, "noParse", false, "Disable EJS parsing")
	startCmd.Flags().StringVar(&logConfig, "log", "", "JSON logging configuration")
	startCmd.Flags().StringVar(&impostersRepo, "impostersRepository", "", "Custom imposters repository")

	// Stop command
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the mountebank server",
		Run:   runStop,
	}

	stopCmd.Flags().StringVar(&pidFile, "pidfile", "mb.pid", "PID file location")

	// Restart command
	restartCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart the mountebank server",
		Run:   runRestart,
	}

	restartCmd.Flags().IntVar(&port, "port", 2525, "Port to run the server on")
	restartCmd.Flags().StringVar(&host, "host", "localhost", "Host to bind to")
	restartCmd.Flags().StringVar(&logLevel, "loglevel", "info", "Log level (debug, info, warn, error)")
	restartCmd.Flags().BoolVar(&allowInjection, "allowInjection", false, "Allow JavaScript injection")
	restartCmd.Flags().StringVar(&pidFile, "pidfile", "mb.pid", "PID file location")
	restartCmd.Flags().StringVar(&configFile, "configfile", "", "Configuration file to load")

	// Save command
	saveCmd := &cobra.Command{
		Use:   "save",
		Short: "Save current imposters to a file",
		Run:   runSave,
	}

	saveCmd.Flags().IntVar(&port, "port", 2525, "Mountebank server port")
	saveCmd.Flags().StringVar(&host, "host", "localhost", "Mountebank server host")
	saveCmd.Flags().StringVar(&saveFile, "savefile", "mb.json", "File to save to")

	// Replay command
	replayCmd := &cobra.Command{
		Use:   "replay",
		Short: "Replay imposters from a file",
		Run:   runReplay,
	}

	replayCmd.Flags().IntVar(&port, "port", 2525, "Mountebank server port")
	replayCmd.Flags().StringVar(&host, "host", "localhost", "Mountebank server host")
	replayCmd.Flags().StringVar(&configFile, "configfile", "", "Configuration file to replay")

	rootCmd.AddCommand(startCmd, stopCmd, restartCmd, saveCmd, replayCmd)

	// Default to start if no command provided
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "start")
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runStart(cmd *cobra.Command, args []string) {
	// Warn about unimplemented flags
	if protoFile != "protocols.json" {
		fmt.Println("Warning: --protofile is not yet implemented")
	}
	if rcFile != "" {
		fmt.Println("Warning: --rcfile is not yet implemented")
	}
	if formatter != "" {
		fmt.Println("Warning: --formatter is not yet implemented")
	}
	if noParse {
		fmt.Println("Warning: --noParse is not yet implemented")
	}
	if logConfig != "" {
		fmt.Println("Warning: --log is not yet implemented")
	}

	var whitelist []string
	if localOnly {
		whitelist = []string{"127.0.0.1", "::1"}
	} else if ipWhitelist != "*" {
		whitelist = strings.Split(ipWhitelist, "|")
	} else {
		whitelist = []string{"*"}
	}

	serverConfig := &server.Config{
		Port:           port,
		Host:           host,
		LogLevel:       logLevel,
		AllowInjection: allowInjection,
		IPWhitelist:    whitelist,
		LogFile:        logFile,
		NoLogFile:      noLogFile,
		Datadir:        datadir,
		Origin:         origin,
		APIKey:         apiKey,
		Debug:          debug,
		LocalOnly:      localOnly,
		ProtoFile:      protoFile,
		Formatter:      formatter,
		NoParse:        noParse,
		LogConfig:      logConfig,
		ImpostersRepo:  impostersRepo,
	}

	srv, err := server.New(serverConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating server: %v\n", err)
		os.Exit(1)
	}

	// Write PID file
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing PID file: %v\n", err)
	}

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		
		if err := srv.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "Error stopping server: %v\n", err)
		}

		// Remove PID file
		os.Remove(pidFile)
		
		os.Exit(0)
	}()

	// Load config file if specified
	// Load config file if specified
	if configFile != "" {
		cfg, err := config.Load(configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config file: %v\n", err)
			os.Exit(1)
		}

		for _, imposterConfig := range cfg.Imposters {
			if err := srv.CreateImposter(&imposterConfig); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating imposter from config: %v\n", err)
				os.Exit(1)
			}
		}
		fmt.Printf("Loaded %d imposters from %s\n", len(cfg.Imposters), configFile)
	}

	// Start server
	if err := srv.Start(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Remove(pidFile)
		os.Exit(1)
	}
}

func runStop(cmd *cobra.Command, args []string) {
	// Read PID file
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading PID file: %v\n", err)
		os.Exit(1)
	}

	var pid int
	fmt.Sscanf(string(pidData), "%d", &pid)

	// Send SIGTERM to process
	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding process: %v\n", err)
		os.Exit(1)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		fmt.Fprintf(os.Stderr, "Error stopping process: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Server stopped")
}

func runRestart(cmd *cobra.Command, args []string) {
	runStop(cmd, args)
	runStart(cmd, args)
}

func runSave(cmd *cobra.Command, args []string) {
	url := fmt.Sprintf("http://%s:%d/imposters?replayable=true&removeProxies=true", host, port)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to mountebank: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Error getting imposters: %s\n", resp.Status)
		os.Exit(1)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(1)
	}

	// The API returns { "imposters": [...] } which matches our Config struct
	// However, we need to unmarshal it to ensure it's valid and then save it
	// Or we can just write the body to file if we trust it
	// But let's use our config package to be safe and consistent

	// We need to unmarshal into a temporary struct because the API response might have extra fields
	// or slightly different structure than what we want to save?
	// Actually, config.Config matches the expected structure.
	
	// Let's just write the body to file for now as it's the simplest way to "save" what the server gave us
	// But wait, config.Save takes []models.ImposterConfig.
	// So we should decode and then save.

	var cfg config.Config
	if err := json.Unmarshal(body, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	if err := config.Save(saveFile, cfg.Imposters); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Saved %d imposters to %s\n", len(cfg.Imposters), saveFile)
}

func runReplay(cmd *cobra.Command, args []string) {
	// Replay is essentially start with a config file
	// But if the server is already running, we might want to just post the imposters?
	// The JS version of replay seems to just start the server with the config.
	// "mb replay" restarts the server with the saved config.
	
	// For now, we'll just treat it as start with config file
	if configFile == "" {
		fmt.Fprintln(os.Stderr, "Error: --configfile is required for replay")
		os.Exit(1)
	}
	
	runStart(cmd, args)
}
