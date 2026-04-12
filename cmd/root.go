package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"muxi/internal/discovery"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:           "muxi",
	Short:         "A tmux project orchestrator in Go",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		if len(args) == 0 {
			if _, ok := discovery.LocalProject(cwd); ok {
				return runLocal(cmd, nil)
			}
			return cmd.Help()
		}

		name := args[0]
		if isReservedCommand(name) {
			return cmd.Help()
		}

		if _, ok, err := discovery.ProjectByName(name); err != nil {
			return err
		} else if ok {
			return runStart(cmd, []string{name})
		}

		return fmt.Errorf("unknown command or project %q", name)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "muxi config file")
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	rootCmd.AddCommand(newStartCommand())
	rootCmd.AddCommand(newStopCommand())
	rootCmd.AddCommand(newStopAllCommand())
	rootCmd.AddCommand(newLocalCommand())
	rootCmd.AddCommand(newDebugCommand())
	rootCmd.AddCommand(newListCommand())
	rootCmd.AddCommand(newNewCommand())
	rootCmd.AddCommand(newCopyCommand())
	rootCmd.AddCommand(newDeleteCommand())
	rootCmd.AddCommand(newImplodeCommand())
	rootCmd.AddCommand(newDoctorCommand())
	rootCmd.AddCommand(newVersionCommand())
	rootCmd.AddCommand(newCommandsCommand())
	rootCmd.AddCommand(newCompletionsCommand())
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".muxi")
	}

	viper.SetEnvPrefix("MUXI")
	viper.AutomaticEnv()
	_ = viper.ReadInConfig()
}

func isReservedCommand(name string) bool {
	reserved := map[string]struct{}{
		"commands": {}, "completions": {}, "copy": {}, "c": {}, "cp": {},
		"debug": {}, "delete": {}, "d": {}, "rm": {}, "doctor": {},
		"help": {}, "implode": {}, "i": {}, "list": {}, "l": {}, "ls": {},
		"local": {}, ".": {}, "new": {}, "open": {}, "edit": {}, "o": {}, "e": {}, "n": {},
		"start": {}, "s": {}, "stop": {}, "st": {}, "stop-all": {}, "version": {}, "-v": {},
	}
	_, ok := reserved[name]
	return ok
}
