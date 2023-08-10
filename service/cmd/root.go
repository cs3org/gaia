// Copyright 2018-2023 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package cmd

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"

	"github.com/cs3org/gaia/service"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	log    *zerolog.Logger
	config *Config
)

var globalFlags = struct {
	Config string
}{}

var rootCmd = &cobra.Command{
	Use:   "gaiasvc",
	Short: "Expose gaia as an HTTP service.",
	Long:  "An HTTP service to make it easy to create custom build of reva.",
	Run: func(cmd *cobra.Command, args []string) {
		config.Gaia.Log = log
		b, err := service.New(&config.Gaia)
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		server := http.Server{
			Addr: config.HTTP.Address,
		}

		// TODO: add support to TLS
		trapSignals(&server, b)
		if err := http.ListenAndServe(config.HTTP.Address, b.Handler()); err != nil {
			log.Fatal().Err(err).Send()
		}
	},
}

func trapSignals(server *http.Server, closable ...io.Closer) {
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		if err := server.Shutdown(context.Background()); err != nil {
			log.Error().Err(err).Msg("error shutting down http server")
		}
		for _, c := range closable {
			if err := c.Close(); err != nil {
				log.Error().Err(err).Send()
			}
		}
	}()
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&globalFlags.Config, "config", "c", "/etc/gaia/config.toml", "config path")
	cobra.OnInitialize(func() {
		initConfig(globalFlags.Config)
		initLogger(&config.Log)
	})

	err := viper.BindPFlags(rootCmd.Flags())
	cobra.CheckErr(err)
}

// func initLogger(config *LogConfig) {
// 	level, err := zerolog.ParseLevel(config.Level)
// 	if err != nil {
// 		level = zerolog.InfoLevel
// 	}

// 	out := zerolog.ConsoleWriter{Out: os.Stderr}
// 	l := zerolog.New(os.Stderr).
// 		With().
// 		Timestamp().
// 		Logger().
// 		Level(level).
// 		Output(out)
// 	log = &l
// }
