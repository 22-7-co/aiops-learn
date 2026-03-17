/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// helloCmd represents the hello command
var helloCmd = &cobra.Command{
	Use:   "hello",
	Short: "say hello to sb.",
	Long: `example: 
	1. ./k8sCopilot he`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("hello called " + args[0])
		fmt.Println("kubeconfig: " + kubeconfig)
		fmt.Println("namespace: " + namespace)
	},
}

var source string

func init() {
	rootCmd.AddCommand(helloCmd)

	helloCmd.Flags().StringVarP(&source, "source", "s", "default", "The source to use")
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// helloCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// helloCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
