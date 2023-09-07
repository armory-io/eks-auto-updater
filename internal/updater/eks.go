package updater

import (
	"context"

	"github.com/armory-io/eks-auto-updater/pkg/aws/eks"
)

type EKSUpdater struct {
	client eks.Interface
}

func NewEKSUpdater(client eks.Interface) *EKSUpdater {
	return &EKSUpdater{
		client: client,
	}
}

// UpdateClusterNodeGroup updates the nodegroup of a cluster to the latest version
func (u EKSUpdater) UpdateClusterNodeGroup(ctx context.Context, clusterName *string, nodegroupName *string, waitForNodeUpdates int) error {
	err := u.client.UpdateNodegroupVersion(ctx, clusterName, nodegroupName, waitForNodeUpdates)
	if err != nil {
		return err
	}

	return nil
}

// UpdateAddon updates the addon of a cluster to the latest version
func (u EKSUpdater) UpdateAddon(ctx context.Context, clusterName *string, addonName *string) (err error) {
	err = u.client.UpdateAddon(ctx, clusterName, addonName)
	if err != nil {
		return err
	}

	return nil
}
