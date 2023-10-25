package eks

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/hashicorp/go-version"
)

type Interface interface {
	// UpdateNodegroupVersion checks if a nodegroup is in an updatable state and
	// triggers an update if it is, then waits for the update to complete
	UpdateNodegroupVersion(ctx context.Context, clusterName, nodegroupName, latestReleaseVersion *string, maxWaitDur int) error

	// UpdateAddon updates the addon of a cluster to the latest version
	UpdateAddon(ctx context.Context, clusterName, addonName *string) error

	// GetAddonsList returns a list of addons in a cluster
	GetAddonsList(ctx context.Context, clusterName *string) ([]string, error)

	// GetClusterVersion returns the kubernetes version of a cluster
	GetClusterVersion(ctx context.Context, clusterName *string) (string, error)
}

type Client struct {
	eks *eks.Client
}

func NewFromConfig(cfg aws.Config) (Interface, error) {
	c := &Client{}
	c.eks = eks.NewFromConfig(cfg)

	return c, nil
}

func (c Client) UpdateNodegroupVersion(ctx context.Context, clusterName, nodegroupName, latestReleaseVersion *string, maxWaitDur int) error {
	// Check if there is an update in progress
	status, err := c.getNodeGroupStatus(ctx, clusterName, nodegroupName)
	if err != nil {
		return err
	}

	if status == types.NodegroupStatusUpdating {
		// If it's already updating, just wait for it to finish
		log.Println("INFO: Nodegroup is already updating. Waiting for completion...")
	} else if status == types.NodegroupStatusActive {
		// If it's active, trigger an update
		log.Println("INFO: Nodegroup is active. Triggering update...")
		jobID, err := c.updateNodegroupVersion(ctx, clusterName, nodegroupName, latestReleaseVersion)
		if err != nil {
			return err
		}
		log.Println("INFO: Upgrade job started... ", jobID)
	} else {
		log.Printf("INFO: Nodegroup is in '%s' state which cannot be updated. Skipping...\n", status)
	}

	// Wait for the update to complete
	err = c.nodegroupUpdateWaiter(ctx, clusterName, nodegroupName, latestReleaseVersion, maxWaitDur)
	if err != nil {
		return err
	}
	log.Println("INFO: Nodegroup update complete")

	return nil
}

// getNodeGroupStatus returns the status of a nodegroup
func (c Client) getNodeGroupStatus(ctx context.Context, clusterName *string, nodegroupName *string) (types.NodegroupStatus, error) {
	// Check if there is an update in progress
	currentNodeGroup, err := c.eks.DescribeNodegroup(ctx, &eks.DescribeNodegroupInput{
		ClusterName:   clusterName,
		NodegroupName: nodegroupName,
	})
	if err != nil {
		return "", fmt.Errorf("ERROR: Unable to describe nodegroup: %w", err)
	}

	fmt.Printf("version: %s\n", *currentNodeGroup.Nodegroup.ReleaseVersion)

	return currentNodeGroup.Nodegroup.Status, nil
}

// getNodeGroupReleaseVersion returns the release version of a nodegroup
func (c Client) getNodeGroupReleaseVersion(ctx context.Context, clusterName, nodegroupName *string) (string, error) {
	currentNodeGroup, err := c.eks.DescribeNodegroup(ctx, &eks.DescribeNodegroupInput{
		ClusterName:   clusterName,
		NodegroupName: nodegroupName,
	})
	if err != nil {
		return "", fmt.Errorf("ERROR: Unable to describe nodegroup: %w", err)
	}

	return *currentNodeGroup.Nodegroup.ReleaseVersion, nil
}

// updateNodegroupVersion updates the nodegroup of a cluster to the latest version
// and returns the ID of the update job
func (c Client) updateNodegroupVersion(ctx context.Context, clusterName, nodegroupName, latestReleaseVersion *string) (string, error) {
	// Trigger an update
	version, err := c.eks.UpdateNodegroupVersion(ctx, &eks.UpdateNodegroupVersionInput{
		ClusterName:    clusterName,
		NodegroupName:  nodegroupName,
		ReleaseVersion: latestReleaseVersion,
	})
	if err != nil {
		return "", fmt.Errorf("ERROR: Update call failed %w", err)
	}

	return *version.Update.Id, nil
}

// nodegroupUpdateWaiter waits for the nodegroup of a cluster to finish updating
// and determines if the update was successful.
func (c Client) nodegroupUpdateWaiter(ctx context.Context, clusterName, nodegroupName, latestReleaseVersion *string, maxWaitDur int) error {
	waiter := eks.NewNodegroupActiveWaiter(c.eks)
	err := waiter.Wait(ctx, &eks.DescribeNodegroupInput{
		ClusterName:   clusterName,
		NodegroupName: nodegroupName,
	},
		time.Duration(maxWaitDur)*time.Minute,
	)
	if err != nil {
		return fmt.Errorf("ERROR: Update failed to complete in the allotted time: %w", err)
	}

	// After the update is complete, check if the version is correct
	currentVersion, err := c.getNodeGroupReleaseVersion(ctx, clusterName, nodegroupName)
	if err != nil {
		return err
	}
	if currentVersion != *latestReleaseVersion {
		return fmt.Errorf("ERROR: Update failed. Expected version %s, got %s", *latestReleaseVersion, currentVersion)
	}

	return nil
}

func (c Client) GetClusterVersion(ctx context.Context, clusterName *string) (string, error) {
	clusterInfo, err := c.eks.DescribeCluster(ctx, &eks.DescribeClusterInput{
		Name: clusterName,
	})
	if err != nil {
		return "", fmt.Errorf("ERROR: Unable to describe cluster: %w", err)
	}

	return *clusterInfo.Cluster.Version, nil
}

func (c Client) GetAddonsList(ctx context.Context, clusterName *string) ([]string, error) {
	addons, err := c.eks.ListAddons(ctx, &eks.ListAddonsInput{
		ClusterName: clusterName,
	})
	if err != nil {
		return nil, fmt.Errorf("ERROR: Unable to list addons: %w", err)
	}

	return addons.Addons, nil
}

func (c Client) UpdateAddon(ctx context.Context, clusterName *string, addonName *string) (err error) {
	addonInfo, err := c.eks.DescribeAddon(ctx, &eks.DescribeAddonInput{
		AddonName:   addonName,
		ClusterName: clusterName,
	})
	if err != nil {
		return fmt.Errorf("ERROR: Unable to describe addon "+*addonName+" in the cluster: %w", err)
	}

	k8sVersion, err := c.GetClusterVersion(ctx, clusterName)
	if err != nil {
		return err
	}
	versions, err := c.eks.DescribeAddonVersions(ctx, &eks.DescribeAddonVersionsInput{
		AddonName:         addonName,
		KubernetesVersion: &k8sVersion,
	})
	if err != nil {
		return fmt.Errorf("ERROR: Failure getting addon versions: %w", err)
	}

	// Find the default version.  TECHNICALLY this is a paginated call, so need to long term add support for that.
	var defaultVersion = ""
	for _, addon := range versions.Addons {
		for _, addonVersion := range addon.AddonVersions {
			for _, capability := range addonVersion.Compatibilities {
				if capability.DefaultVersion {
					defaultVersion = *addonVersion.AddonVersion
				}
			}
		}
	}
	if len(defaultVersion) == 0 {
		return fmt.Errorf("ERROR: Unable to find default version for addon "+*addonName+" in the cluster: %w", err)
	}

	currentVersion, err := version.NewVersion(*addonInfo.Addon.AddonVersion)
	if err != nil {
		return fmt.Errorf("ERROR: Unable to parse version correctly for addon "+*addonName+" version of "+*addonInfo.Addon.AddonVersion+", %w", err)
	}
	newVersion, err := version.NewVersion(defaultVersion)
	if err != nil {
		return fmt.Errorf("ERROR: Unable to parse version correctly for addon "+*addonName+" version of "+defaultVersion+", %w", err)
	}
	if newVersion.Compare(currentVersion) < 0 {
		log.Println("WARNING: Skipping addon " + *addonName + " as the version to upgrade to is older/equal to the version installed!")
		return nil
	}
	log.Println("INFO: Updating addon " + *addonName + " from: " + *addonInfo.Addon.AddonVersion + " to:" + defaultVersion)

	// NOMINALLY we should check if there's a service account/config and apply that here not just default to node settings :)
	response, err := c.eks.UpdateAddon(ctx, &eks.UpdateAddonInput{
		AddonName:        addonName,
		ClusterName:      clusterName,
		AddonVersion:     &defaultVersion,
		ResolveConflicts: "OVERWRITE",
	})
	if err != nil {
		return err
	}

	log.Println("INFO: Addon update triggered ... ID of  " + *response.Update.Id + "... waiting for completion")

	err = eks.NewAddonActiveWaiter(c.eks).Wait(ctx, &eks.DescribeAddonInput{
		AddonName:   addonName,
		ClusterName: clusterName,
	}, time.Duration(20)*time.Minute)
	if err != nil {
		return fmt.Errorf("ERROR: Update failed to complete in the allotted time: %w", err)
	}

	log.Printf("INFO: Addon %s update complete\n", *addonName)

	return err
}
