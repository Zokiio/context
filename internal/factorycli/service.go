package factorycli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/Zokiio/context/internal/factory"
	"github.com/Zokiio/context/internal/factoryhttp"
	cli "github.com/urfave/cli/v3"
)

func initCommand(out io.Writer) *cli.Command {
	return &cli.Command{Name: "init", Usage: "Bootstrap the first administrator into a private token file", Flags: []cli.Flag{&cli.StringFlag{Name: "data-dir", Required: true}, &cli.StringFlag{Name: "username", Value: "admin"}}, Action: func(ctx context.Context, c *cli.Command) error {
		path := c.String("token-file")
		f, e := reserveSecret(path)
		if e != nil {
			return e
		}
		defer f.Close()
		ok := false
		defer func() {
			if !ok {
				os.Remove(path)
			}
		}()
		service, e := factory.Open(c.String("data-dir"), factory.Options{})
		if e != nil {
			return e
		}
		defer service.Close()
		result, e := service.Bootstrap(ctx, factory.BootstrapRequest{Username: c.String("username")})
		if e != nil {
			return e
		}
		if e = storeSecret(f, result.Token); e != nil {
			return e
		}
		ok = true
		return encode(out, map[string]any{"user": result.User, "tokenFile": path})
	}}
}
func serveCommand(stderr io.Writer) *cli.Command {
	return &cli.Command{Name: "serve", Usage: "Run the durable service; non-loopback listeners require TLS", Flags: append([]cli.Flag{&cli.StringFlag{Name: "data-dir", Required: true}, &cli.StringFlag{Name: "listen", Value: "127.0.0.1:8741"}, &cli.StringFlag{Name: "tls-cert"}, &cli.StringFlag{Name: "tls-key"}, &cli.BoolFlag{Name: "pilot", Usage: "Run the autopilot, which carries out go-aheads and saved ask policies"}, &cli.DurationFlag{Name: "pilot-interval", Value: 5 * time.Second}}, append(append(decisionFlags(), refinerFlags()...), expectationFlags()...)...), Action: func(ctx context.Context, c *cli.Command) error {
		address := c.String("listen")
		cert, key := c.String("tls-cert"), c.String("tls-key")
		if e := validateListener(address, cert, key); e != nil {
			return e
		}
		options := serviceOptions(c)
		options.Pilot, options.PilotInterval = c.Bool("pilot"), c.Duration("pilot-interval")
		refiner, e := refinerCommandFile(c)
		if e != nil {
			return e
		}
		options.RefinerCommand = refiner
		if options.ExpectationCommand, e = commandFile(c.String("expectation-command-file")); e != nil {
			return e
		}
		service, e := factory.Open(c.String("data-dir"), options)
		if e != nil {
			return e
		}
		defer service.Close()
		listener, e := net.Listen("tcp", address)
		if e != nil {
			return e
		}
		defer listener.Close()
		server := &http.Server{Handler: factoryhttp.NewHandler(service), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
		done := make(chan error, 1)
		go func() {
			if cert != "" {
				done <- server.ServeTLS(listener, cert, key)
			} else {
				done <- server.Serve(listener)
			}
		}()
		scheme := "http"
		if cert != "" {
			scheme = "https"
		}
		fmt.Fprintf(stderr, "Waymark factory listening at %s://%s\n", scheme, listener.Addr())
		select {
		case e = <-done:
			if errors.Is(e, http.ErrServerClosed) {
				return nil
			}
			return e
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if e = server.Shutdown(shutdownCtx); e != nil {
				return e
			}
			e = <-done
			if errors.Is(e, http.ErrServerClosed) {
				return nil
			}
			return e
		}
	}}
}
func validateListener(address, cert, key string) error {
	host, _, e := net.SplitHostPort(address)
	if e != nil {
		return fmt.Errorf("invalid listener address: %w", e)
	}
	if (cert == "") != (key == "") {
		return errors.New("both tls-cert and tls-key are required")
	}
	ip := net.ParseIP(host)
	loopback := ip != nil && ip.IsLoopback()
	if !loopback && cert == "" {
		return errors.New("non-loopback listener requires tls-cert and tls-key; use 127.0.0.1 for local HTTP")
	}
	return nil
}
