package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	chclient "github.com/jpillora/chisel/client"
	chshare "github.com/jpillora/chisel/share"
	"github.com/jpillora/chisel/share/cos"
)

func main() {
	flags := flag.NewFlagSet("agent", flag.ContinueOnError)
	config := chclient.Config{Headers: http.Header{}}

	// Define all flags
	flags.StringVar(&config.Fingerprint, "fingerprint", "", "")
	flags.StringVar(&config.Auth, "auth", "", "")
	flags.DurationVar(&config.KeepAlive, "keepalive", 25*time.Second, "")
	flags.IntVar(&config.MaxRetryCount, "max-retry-count", -1, "")
	flags.DurationVar(&config.MaxRetryInterval, "max-retry-interval", 0, "")
	flags.StringVar(&config.Proxy, "proxy", "", "")
	flags.StringVar(&config.TLS.CA, "tls-ca", "", "")
	flags.BoolVar(&config.TLS.SkipVerify, "tls-skip-verify", false, "")
	flags.StringVar(&config.TLS.Cert, "tls-cert", "", "")
	flags.StringVar(&config.TLS.Key, "tls-key", "", "")
	flags.Var(&headerFlags{config.Headers}, "header", "")
	hostname := flags.String("hostname", "", "")
	sni := flags.String("sni", "", "")
	verbose := flags.Bool("v", false, "")
	version := flags.Bool("version", false, "")

	flags.Usage = func() {
		fmt.Print("You know what to do")
		os.Exit(0)
	}

	flags.Parse(os.Args[1:])

	// Handle version flag
	if *version {
		fmt.Println(chshare.BuildVersion)
		os.Exit(0)
	}

	// Get remaining args (server and remotes)
	args := flags.Args()
	if len(args) < 2 {
		log.Fatalf("Server/Remote required")
	}

	config.Server = args[0]
	config.Remotes = args[1:]

	// Default auth from environment
	if config.Auth == "" {
		config.Auth = os.Getenv("AUTH")
	}

	// Move hostname onto headers
	if *hostname != "" {
		config.Headers.Set("Host", *hostname)
		config.TLS.ServerName = *hostname
	}

	if *sni != "" {
		config.TLS.ServerName = *sni
	}

	// Create and start client
	c, err := chclient.NewClient(&config)
	if err != nil {
		log.Fatal(err)
	}
	c.Debug = *verbose

	go cos.GoStats()
	ctx := cos.InterruptContext()
	if err := c.Start(ctx); err != nil {
		log.Fatal(err)
	}
	if err := c.Wait(); err != nil {
		log.Fatal(err)
	}
}

type headerFlags struct {
	http.Header
}

func (flag *headerFlags) String() string {
	out := ""
	for k, v := range flag.Header {
		out += fmt.Sprintf("%s: %s\n", k, v)
	}
	return out
}

func (flag *headerFlags) Set(arg string) error {
	index := strings.Index(arg, ":")
	if index < 0 {
		return fmt.Errorf(`Invalid header (%s). Should be in the format "HeaderName: HeaderContent"`, arg)
	}
	if flag.Header == nil {
		flag.Header = http.Header{}
	}
	key := arg[0:index]
	value := arg[index+1:]
	flag.Header.Set(key, strings.TrimSpace(value))
	return nil
}