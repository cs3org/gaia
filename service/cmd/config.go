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
	"github.com/cs3org/gaia/service"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type HTTPConfig struct {
	Address string `mapstructure:"address"`
}

type LogConfig struct {
	Level         string `mapstructure:"level"`
	Output        string `mapstructure:"output"`
	DisableStdout bool   `mapstructure:"disable_stdout"`
}

type Config struct {
	HTTP HTTPConfig     `mapstructure:"http"`
	Log  LogConfig      `mapstructure:"log"`
	Gaia service.Config `mapstructure:"gaia"`
}

func initConfig(file string) {
	viper.SetConfigFile(file)
	err := viper.ReadInConfig()
	cobra.CheckErr(err)

	var cfg Config
	err = mapstructure.Decode(viper.AllSettings(), &cfg)
	cobra.CheckErr(err)
	config = &cfg
}
