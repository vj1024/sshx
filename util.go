package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func printHelp() {
	log.Println(strings.TrimSpace(`
Usage:
  login to remote server by user and host:
        sshx [options] user@host
  login to remote server by alias in config file:
        sshx [options] alias

Options:
`))
	flag.PrintDefaults()
}

func formatPath(path string) string {
	home, _ := os.UserHomeDir()
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}
