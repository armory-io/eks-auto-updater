package cmd

import (
	"errors"
	"strings"
	"sync"

	"github.com/armory-io/eks-auto-updater/internal/updater"
	"github.com/armory-io/eks-auto-updater/pkg/aws"
	"github.com/armory-io/eks-auto-updater/pkg/aws/options"

	"github.com/spf13/cobra"
)

// addonsCmd represents the addons command
var addonsCmd = &cobra.Command{
	Use:   "addons",
	Short: "Upgrades EKS addons to the latest version",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		region := cmd.Flag("region").Value.String()
		roleArn := cmd.Flag("role-arn").Value.String()
		clusterName := cmd.Flag("cluster-name").Value.String()
		addons := cmd.Flag("addons").Value.String()
		addonsList := strings.Split(addons, ",")

		awsClient, err := aws.NewClient(ctx,
			options.WithRegion(region),
			options.WithRoleArn(roleArn),
		)
		if err != nil {
			return err
		}

		var wg sync.WaitGroup
		errChan := make(chan error)

		updater := updater.NewEKSUpdater(awsClient.EKS(), awsClient.SSM())
		for _, addon := range addonsList {
			wg.Add(1)

			go func(addon string, wg *sync.WaitGroup) {
				defer wg.Done()

				errChan <- updater.UpdateAddon(ctx, &clusterName, &addon)
			}(addon, &wg)
		}

		go func() {
			wg.Wait()
			close(errChan)
		}()

		var errResult error
		for err := range errChan {
			if err != nil {
				errResult = errors.Join(errResult, err)
			}
		}

		// Return all of the errors that were sent to the errChan channel
		if errResult != nil {
			return errResult
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(addonsCmd)

	addonsCmd.Flags().StringP("addons", "a", "kube-proxy,vpc-cni,coredns,aws-ebs-csi-driver", "Comma separated list of addons to update. For example: kube-proxy,vpc-cni,coredns,aws-ebs-csi-driver. Defaults to all addons")
}
