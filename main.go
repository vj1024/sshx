package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/creack/pty"
	flag "github.com/jessevdk/go-flags"
	"golang.org/x/term"
)

// Version is the program version set on build.
var Version = "unknown"

type options struct {
	Port           uint          `short:"p" long:"port" description:"Show verbose debug information"`
	Log            string        `short:"l" long:"log" description:"The file to write error logs, default to stderr"`
	Config         string        `short:"c" long:"config" description:"The config file to load, in yaml format. If not specified, the file path is read from the environment variable 'SSHX_CONFIG' or else '~/.ssh/sshx.yaml'"`
	IdleMaxTime    time.Duration `short:"i" long:"idle-time" description:"The max idle time, when reaching this time, send the idle string to shell automatically"`
	IdleSendString string        `short:"s" long:"idle-string" description:"The string to send when shell is idle"`
	PrintAlias     bool          `short:"a" long:"alias" description:"Print all aliases"`
	PrintVersion   bool          `short:"v" long:"version" description:"Print version"`
}

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stderr)

	var opt options
	parser := flag.NewParser(&opt, flag.Default)
	parser.Usage = `[OPTIONS] destination

  login to remote server by user and host:
    sshx [OPTIONS] user@host

  login to remote server by alias in config file:
    sshx [OPTIONS] alias`
	args, err := parser.Parse()
	if err != nil {
		return
	}

	if opt.PrintVersion {
		fmt.Println(Version)
		return
	}

	if opt.Log != "" {
		f, err := os.OpenFile(formatPath(opt.Log), os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			log.Fatalf("open log file err: %v", err)
		}
		defer func() { _ = f.Close() }()
		log.SetOutput(f)
	}

	if opt.Config == "" {
		opt.Config = os.Getenv("SSHX_CONFIG")
	}
	if opt.Config == "" {
		opt.Config = "~/.ssh/sshx.yaml"
	}
	conf, err := loadConfig(formatPath(opt.Config))
	if err != nil {
		log.Fatalf("load config file `%s` err: %v", opt.Config, err)
	}

	if opt.PrintAlias {
		conf.printAliases()
		return
	}

	if len(args) != 1 {
		parser.WriteHelp(os.Stderr)
		os.Exit(1)
	}

	sv, err := conf.findServer(args[0])
	if err != nil {
		log.Fatal(err)
	}

	// Command line args take precedence over configuration from file.
	if opt.IdleMaxTime > 0 {
		sv.IdleMaxSeconds = int(math.Ceil(opt.IdleMaxTime.Seconds()))
	}
	if opt.IdleSendString != "" {
		sv.IdleSendString = opt.IdleSendString
	}
	if opt.Port > 0 {
		sv.Port = opt.Port
	}

	if err := start(sv); err != nil {
		log.Fatal(err)
	}
}

func start(sv *Server) error {
	// Create arbitrary command.
	c := exec.Command("ssh", "-p", fmt.Sprint(sv.Port), fmt.Sprintf("%s@%s", sv.User, sv.Host))
	c.Env = os.Environ()

	// Start the command with a pty.
	ptmx, err := pty.Start(c)
	if err != nil {
		return err
	}
	// Make sure to close the pty at the end.
	defer func() { _ = ptmx.Close() }() // Best effort.

	// Handle pty size.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				log.Printf("error resizing pty: %s", err)
			}
		}
	}()
	ch <- syscall.SIGWINCH                        // Initial resize.
	defer func() { signal.Stop(ch); close(ch) }() // Cleanup signals when done.

	// Set stdin in raw mode.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.

	// Copy stdin to the pty and the pty to stdout.
	// NOTE: The goroutine will keep reading until the next keystroke before returning.

	var r io.Reader
	var w io.Writer

	r = os.Stdin
	w = os.Stdout

	// if enable keepalive
	if sv.IdleMaxSeconds > 0 {
		aliveChan := make(chan struct{})
		go func() {
			idleTime := time.Duration(sv.IdleMaxSeconds) * time.Second
			const minIdleTime = 5 * time.Second
			if idleTime < minIdleTime {
				idleTime = minIdleTime
			}
			if sv.IdleSendString == "" {
				sv.IdleSendString = "@"
			}

			t := time.NewTicker(idleTime)
			defer t.Stop()

			for {
				select {
				case <-aliveChan:
					t.Reset(idleTime)
				case <-t.C:
					ptmx.WriteString(sv.IdleSendString)
				}
			}
		}()
		r = &keepaliveReader{r, aliveChan}
		w = &keepaliveWriter{w, aliveChan}
	}

	go func() {
		if _, err = io.Copy(ptmx, r); err != nil {
			log.Fatalf("copy stdin to ptmx err: %v", err)
		}
	}()

	w = &expectWriter{
		receive: w,
		send:    ptmx,
		server:  sv,
	}
	_, _ = io.Copy(w, ptmx)

	return nil
}
