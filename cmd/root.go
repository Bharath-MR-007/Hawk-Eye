// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewCmdRoot creates a new root command
func NewCmdRoot(version string) *cobra.Command {
	var cfgFile string

	rootCmd := &cobra.Command{
		Use:   "hawkeye",
		Short: "Hawkeye, the infrastructure monitoring agent",
		Long: "Hawkeye is an infrastructure monitoring agent that is able to perform different checks.\n" +
			"The check results are exposed via an API.",
		Version: version,
	}

	cobra.OnInitialize(func() {
		initConfig(cfgFile)
	})

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.hawkeye.yaml)")

	return rootCmd
}

// Execute adds all child commands to the root command
// and executes the cmd tree
func Execute(version string) {
	cmd := BuildCmd(version)

	if err := cmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func BuildCmd(version string) *cobra.Command {
	cmd := NewCmdRoot(version)
	cmd.AddCommand(NewCmdRun())
	return cmd
}

func initConfig(cfgFile string) {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".hawkeye" (without an extension)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".hawkeye")
	}

	viper.SetOptions(viper.ExperimentalBindStruct())
	viper.SetEnvPrefix("hawkeye")
	dotreplacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(dotreplacer)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
