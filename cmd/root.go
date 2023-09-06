package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "eks-auto-updater",
	Short: "Updates EKS cluster components to the latest version",
	Long: `eks-auto-updater is a CLI tool that updates following components of an EKS cluster to the latest version:
	- Nodegroups
	- Addons
	`,
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	// region
	rootCmd.PersistentFlags().String("region", "us-west-2", "AWS Region to use")
	rootCmd.MarkPersistentFlagRequired("region")

	// cluster name
	rootCmd.PersistentFlags().String("cluster-name", "", "Name of the EKS cluster to update")
	rootCmd.MarkPersistentFlagRequired("cluster-name")

	// role arn
	rootCmd.PersistentFlags().String("role-arn", "", "ARN of the role to assume")
}
