package main

import (
	_ "embed"
	"log"
	"os"
	"time"

	"github.com/jiotv-go/jiotv_go/v3/cmd"
	"github.com/jiotv-go/jiotv_go/v3/internal/constants"
	"github.com/jiotv-go/jiotv_go/v3/pkg/secureurl"
	"github.com/jiotv-go/jiotv_go/v3/pkg/store"

	"github.com/urfave/cli/v2"
)

//go:embed VERSION
var version string

func main() {
	constants.Version = version

	app := &cli.App{
		Name:      "JioTV Go",
		Usage:     "Stream JioTV on any device",
		HelpName:  "jiotv_go",
		Version:   version,
		Copyright: "© JioTV Go (https://github.com/jiotv-go/jiotv_go)",
		Compiled:  time.Now(),
		Suggest:   true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "",
				Usage:   "Path to config file",
			},
			&cli.BoolFlag{
				Name:    "skip-update-check",
				Aliases: []string{"skip-update"},
				Usage:   "Skip checking for update on startup",
			},
		},
		Before: func(c *cli.Context) error {
			configPath := c.String("config")
			if err := cmd.LoadConfig(configPath); err != nil {
				log.Fatalf("Failed to load config: %v", err)
			}
			if c.Bool("skip-update-check") {
				log.Println("INFO: Skipping update check")
			} else {
				cmd.PrintIfUpdateAvailable(c)
			}
			cmd.InitializeLogger()
			if err := store.Init(); err != nil {
				return err
			}
			secureurl.Init()
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:        "serve",
				Aliases:     []string{"run", "start"},
				Usage:       "Start JioTV Go server",
				Description: "Starts the JioTV Go server.",
				Action: func(c *cli.Context) error {
					// ✅ THIS IS THE FIX FOR RENDER
					port := os.Getenv("PORT")
					if port == "" {
						port = c.String("port") // fallback if not on Render
					}

					host := c.String("host")
					if c.Bool("public") {
						cmd.Logger().Println("INFO: Public flag set, exposing server")
						host = "[::]"
					} else {
						host = "0.0.0.0" // ✅ REQUIRED FOR RENDER
					}

					tls := c.Bool("tls")
					tlsCertPath := c.String("tls-cert")
					tlsKeyPath := c.String("tls-key")

					return cmd.JioTVServer(cmd.JioTVServerConfig{
						Host:        host,
						Port:        port,
						TLS:         tls,
						TLSCertPath: tlsCertPath,
						TLSKeyPath:  tlsKeyPath,
					})
				},
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "host", Aliases: []string{"H"}, Value: "localhost", Usage: "Host to listen on"},
					&cli.StringFlag{Name: "port", Aliases: []string{"p"}, Value: "5001", Usage: "Port to listen on"},
					&cli.BoolFlag{Name: "public", Aliases: []string{"P"}, Usage: "Expose to public (use [::] as host)"},
					&cli.BoolFlag{Name: "tls", Aliases: []string{"https"}, Usage: "Enable TLS"},
					&cli.StringFlag{Name: "tls-cert", Aliases: []string{"cert"}, Usage: "TLS cert path"},
					&cli.StringFlag{Name: "tls-key", Aliases: []string{"cert-key"}, Usage: "TLS key path"},
				},
			},
			// other commands...
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
