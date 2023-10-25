package updater

import (
	"context"

	"github.com/armory-io/eks-auto-updater/pkg/aws/eks"
	"github.com/armory-io/eks-auto-updater/pkg/aws/ssm"
)

type EKSUpdater struct {
	eksClient eks.Interface
	ssmClient ssm.Interface
}

func NewEKSUpdater(eksClient eks.Interface, ssmClient ssm.Interface) *EKSUpdater {
	return &EKSUpdater{
		eksClient: eksClient,
		ssmClient: ssmClient,
	}
}

// UpdateClusterNodeGroup updates the nodegroup of a cluster to the latest version
func (u EKSUpdater) UpdateClusterNodeGroup(ctx context.Context, clusterName, nodegroupName *string, waitForNodeUpdates int) error {
	// Retrieve the latest AMI release version from SSM parameter store (managed by AWS)
	// and use this information to determine if the update is successful
	version, err := u.eksClient.GetClusterVersion(ctx, clusterName)
	if err != nil {
		return err
	}
	latestVersion, err := u.ssmClient.GetLatestAMIReleaseVersion(ctx, &version, clusterName)
	if err != nil {
		return err
	}

	// Update the nodegroup to the latest version
	err = u.eksClient.UpdateNodegroupVersion(ctx, clusterName, nodegroupName, &latestVersion, waitForNodeUpdates)
	if err != nil {
		return err
	}

	return nil
}

// UpdateAddon updates the addon of a cluster to the latest version
func (u EKSUpdater) UpdateAddon(ctx context.Context, clusterName *string, addonName *string) (err error) {
	err = u.eksClient.UpdateAddon(ctx, clusterName, addonName)
	if err != nil {
		return err
	}

	return nil
}
