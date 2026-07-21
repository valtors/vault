package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/valtors/vault/internal/api"
	"github.com/valtors/vault/internal/sandbox"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version", "-v", "--version":
		fmt.Println(version)
	case "help", "-h", "--help":
		usage()
	case "run":
		runCmd(os.Args[2:])
	case "serve":
		serveCmd(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func runCmd(args []string) {
	fs := flag.NewFlagSet("vault run", flag.ExitOnError)
	timeout := fs.Int("timeout", 0, "timeout in seconds (0 = no limit)")
	dir := fs.String("dir", "", "sandbox root directory")
	allow := fs.String("allow", "", "comma-separated list of allowed directories")
	fs.Parse(args)

	rest := fs.Args()
	if len(rest) == 0 {
		fmt.Fprintln(os.Stderr, "vault run: no command specified")
		os.Exit(1)
	}

	cfg := sandbox.Config{
		Command:     rest[0],
		Args:        rest[1:],
		TimeoutSecs: *timeout,
		RootDir:      *dir,
	}

	if *allow != "" {
		cfg.AllowedDirs = splitCSV(*allow)
	}

	sb, err := sandbox.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vault: %v\n", err)
		os.Exit(1)
	}
	defer sb.Cleanup()

	if err := sb.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "vault: %v\n", err)
		os.Exit(1)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		sb.Kill()
	}()

	if err := sb.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "vault: %v\n", err)
		os.Exit(1)
	}
}

func serveCmd(args []string) {
	fs := flag.NewFlagSet("vault serve", flag.ExitOnError)
	port := fs.Int("port", 9090, "api server port")
	fs.Parse(args)

	srv := api.NewServer(*port)
	fmt.Fprintf(os.Stderr, "vault api on :%d\n", *port)
	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "vault: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `vault %s - sandbox for ai agents

usage:
  vault run [flags] <command> [args...]    run a command in a sandbox
  vault serve [flags]                      start the api server
  vault version                            print version
  vault help                               show this message

run flags:
  -timeout int     timeout in seconds (0 = no limit)
  -dir string      sandbox root directory
  -allow string    comma-separated allowed directories

serve flags:
  -port int        api server port (default 9090)

examples:
  vault run -timeout 60 -- claude-code
  vault run --allow /home/user/project -- python script.py
  vault serve -port 8080
`, version)
}

func splitCSV(s string) []string {
	var result []string
	current := ""
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(s[i])
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
