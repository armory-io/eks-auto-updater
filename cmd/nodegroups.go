package cmd

import (
	"github.com/armory-io/eks-auto-updater/internal/updater"
	"github.com/armory-io/eks-auto-updater/pkg/aws"
	"github.com/armory-io/eks-auto-updater/pkg/aws/options"

	"github.com/spf13/cobra"
)

// nodegroupCmd represents the nodegroup command
var nodegroupsCmd = &cobra.Command{
	Use:   "nodegroups",
	Short: "Upgrades EKS nodegroups to the latest version",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		region := cmd.Flag("region").Value.String()
		roleArn := cmd.Flag("role-arn").Value.String()
		clusterName := cmd.Flag("cluster-name").Value.String()
		nodegroupName := cmd.Flag("nodegroup-name").Value.String()
		waitForNodeUpdates, _ := cmd.Flags().GetInt("nodegroup-wait-time")

		awsClient, err := aws.NewClient(ctx,
			options.WithRegion(region),
			options.WithRoleArn(roleArn),
		)
		if err != nil {
			return err
		}

		updater := updater.NewEKSUpdater(awsClient.EKS())
		err = updater.UpdateClusterNodeGroup(ctx, &clusterName, &nodegroupName, waitForNodeUpdates)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(nodegroupsCmd)

	// Nodegroup Name
	nodegroupsCmd.Flags().String("nodegroup-name", "", "Name of the EKS nodegroup to update")
	nodegroupsCmd.MarkFlagRequired("nodegroup-name")

	// Nodegroup Wait Time
	nodegroupsCmd.Flags().Int("nodegroup-wait-time", 120, "Time in minutes to wait for node group update to complete.  Defaults to 120 minutes")
}
