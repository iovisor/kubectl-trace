package main

import (
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	//"k8s.io/cli-runtime/pkg/genericclioptions"
)

var cfgFile string

//var parentConfigFlags genericclioptions.ConfigFlags

var rootCmd = &cobra.Command{
	Use:   "trace",
	Short: "Execute and manage bpftrace programs on your kubernetes cluster",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kubectl-trace.yaml)")

	// TODO(leodido): figure out how to use the flag from the main kubectl
	// instead of having to recreate them like below
	//parentConfigFlags = genericclioptions.ConfigFlags{}
	//parentConfigFlags.AddFlags(rootCmd.PersistentFlags())

	rootCmd.PersistentFlags().String("kubeconfig", "", "Path to the kubeconfig file to use for CLI requests.")
	viper.BindPFlag("kubeconfig", rootCmd.PersistentFlags().Lookup("kubeconfig"))
	viper.BindEnv("kubeconfig", "KUBECONFIG")
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(deleteCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".kubectl-trace" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".kubectl-trace")
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
