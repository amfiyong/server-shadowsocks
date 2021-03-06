package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/xflash-panda/server-shadowsocks/internal/app/server"
	"github.com/xflash-panda/server-shadowsocks/internal/pkg/api"
	"github.com/xflash-panda/server-shadowsocks/internal/pkg/service"
	"github.com/xtls/xray-core/core"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

const (
	Name          = "shadowsocks-node"
	Version       = "0.1.0"
	CopyRight     = "XFLASH-PANDA@2021"
	LogLevelDebug = "debug"
	LogLevelError = "error"
	LogLevelInfo  = "info"
)

func init() {
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"V"},
		Usage:   "print only the version",
	}
	cli.ErrWriter = ioutil.Discard

	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("version=%s xray.version=%s\n", Version, core.Version())
	}
}

func main() {
	var config server.Config
	var apiConfig api.Config
	var serviceConfig service.Config

	app := &cli.App{
		Name:      Name,
		Version:   Version,
		Copyright: CopyRight,
		Usage:     "Provide shadowsocks service for the v2Board(XFLASH-PANDA)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "api",
				Usage:       "Server address",
				EnvVars:     []string{"X_PANDA_SS_API", "API"},
				Required:    true,
				Destination: &apiConfig.APIHost,
			},
			&cli.StringFlag{
				Name:        "token",
				Usage:       "Token of server API",
				EnvVars:     []string{"X_PANDA_SS_TOKEN", "TOKEN"},
				Required:    true,
				Destination: &apiConfig.Token,
			},
			&cli.IntFlag{
				Name:        "node",
				Usage:       "Node ID",
				EnvVars:     []string{"X_PANDA_SS_NODE", "NODE"},
				Required:    true,
				Destination: &apiConfig.NodeID,
			},
			&cli.DurationFlag{
				Name:        "sys_interval",
				Usage:       "API request cycle, unit: second",
				EnvVars:     []string{"X_PANDA_SS_SYS_INTERVAL", "SYS_INTERVAL"},
				Value:       time.Second * 60,
				DefaultText: "60",
				Required:    false,
				Destination: &serviceConfig.SysInterval,
			},

			&cli.StringFlag{
				Name:        "log_mode",
				Value:       LogLevelError,
				Usage:       "Log mode",
				EnvVars:     []string{"X_PANDA_SS_LOG_LEVEL", "LOG_LEVEL"},
				Destination: &config.LogLevel,
				Required:    false,
			},
		},
		Before: func(c *cli.Context) error {
			log.SetFormatter(&log.TextFormatter{})
			if config.LogLevel == LogLevelDebug {
				log.SetFormatter(&log.TextFormatter{
					FullTimestamp: true,
				})
				log.SetLevel(log.DebugLevel)
				log.SetReportCaller(true)
			} else if config.LogLevel == LogLevelInfo {
				log.SetLevel(log.InfoLevel)
			} else if config.LogLevel == LogLevelError {
				log.SetLevel(log.ErrorLevel)
			} else {
				return fmt.Errorf("log mode %s not supported", config.LogLevel)
			}
			return nil
		},
		Action: func(c *cli.Context) error {
			if config.LogLevel != LogLevelDebug {
				defer func() {
					if r := recover(); r != nil {
						log.Fatal(r)
					}
				}()
			}
			serv := server.New(&config, &apiConfig, &serviceConfig)
			serv.Start()
			defer serv.Close()
			runtime.GC()
			{
				osSignals := make(chan os.Signal, 1)
				signal.Notify(osSignals, os.Interrupt, os.Kill, syscall.SIGTERM)
				<-osSignals
			}
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
