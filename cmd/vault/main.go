package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/valtors/vault/internal/api"
	"github.com/valtors/vault/internal/sandbox"
	"github.com/valtors/vault/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runCmd(os.Args[2:])
	case "serve":
		serveCmd(os.Args[2:])
	case "version":
		fmt.Println("vault 0.1.0")
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "vault: unknown command %q\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`vault

run your agent. it can't destroy your machine.

commands:
  run <cmd> [args...]   run a command inside the sandbox
  serve [flags]         start the HTTP API server
  version               print version

flags for serve:
  -port <n>   listen port (default 9090)

examples:
  vault run -- claude-code
  vault run -- npx -y @modelcontextprotocol/server-filesystem /tmp
  vault serve -port 8080`)
}

func runCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "vault: run needs a command")
		os.Exit(1)
	}

	db, err := store.New("/tmp/vault-audit.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "vault: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	cfg := sandbox.DefaultConfig()
	sb := sandbox.New(cfg, db)

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		sb.Stop()
	}()

	if err := sb.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "vault: %v\n", err)
		os.Exit(1)
	}
}

func serveCmd(args []string) {
	port := 9090
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-port":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &port)
				i++
			}
		}
	}

	db, err := store.New("/tmp/vault-audit.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "vault: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	srv := api.NewServer(db, port)
	fmt.Printf("vault listening on :%d\n", port)
	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "vault: %v\n", err)
		os.Exit(1)
	}
}
