package main

import (
	"context"
	"flag"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"log"
	"strings"
	"time"
)

/*
	LONG TERM goals:
	Update CLUSTER VERSION as well.   E.g. this can update the whole shebang saving a lot of time/headache.  We can validate things beforehand LIKE
		- Make sure there are PDBs on resources like spinnaker first
		- Verify nodes are on an n-1 release version first.  AND update nodes post upgrade kinda things
*/
func main() {

	clusterName := flag.String("cluster-name", "", "Cluster name REQUIRED")
	//TODO: LOOK UP managed node groups instead of parameters... enhancement for later.  AND update multiple node groups sequentially would be a later thing
	nodegroupName := flag.String("nodegroup-name", "", "Node group name to update REQUIRED")
	waitTimeForNodeUpdates := *flag.Int("nodegroup-wait-time", 120, "Time in minutes to wait for node group update to complete.  Defaults to 120 minutes")
	addonsToUpdate := strings.Split(*flag.String("addons-to-update", "kube-proxy,coredns,vpc-cni,aws-ebs-csi-driver", "Comma separated list of adds on to updates.  Defaults to kube-proxy, coredns, vpc-cni, aws-ebs-csi-driver addons"), ",")
	flag.Parse()
	if len(*clusterName) == 0 {
		log.Fatal("Invalid cluster name!  Must be set!")
	}

	// Load the Shared AWS Configuration (~/.aws/config)
	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal("ERROR: Unable to auth/get connected to AWS", err)
	}
	client := eks.NewFromConfig(cfg)

	clusterInformation, _ := client.DescribeCluster(ctx, &eks.DescribeClusterInput{Name: clusterName})
	if len(*nodegroupName) == 0 {
		// Lookup and update the node groups...
		nodeGroups, nodeGroupListErr := client.ListNodegroups(ctx, &eks.ListNodegroupsInput{ClusterName: clusterName})
		if nodeGroupListErr != nil {
			log.Fatal("ERROR: Unable to list node groups... ", nodeGroupListErr)
		}
		for _, nodeGroup := range nodeGroups.Nodegroups {
			log.Println("INFO: Starting updates of node group " + nodeGroup)
			updateError := updateClusterNodeGroup(client, ctx, clusterName, &nodeGroup, waitTimeForNodeUpdates)
			if updateError != nil {
				log.Fatal("ERROR: Unable to update cluster node group... ", updateError)
			}
		}
	} else {
		log.Println("INFO: Starting updates of node group " + *nodegroupName)
		updateError := updateClusterNodeGroup(client, ctx, clusterName, nodegroupName, waitTimeForNodeUpdates)
		if updateError != nil {
			log.Fatal("ERROR: Unable to update cluster node group... ", updateError)
		}
	}
	for _, addon := range addonsToUpdate {
		updateAddon(client, ctx, clusterName, &addon, clusterInformation.Cluster.Version)
	}
	log.Println("INFO:  Updates complete!")

}

func updateAddon(client *eks.Client, ctx context.Context, clusterName *string, addonName *string, k8sVersion *string) {
	versions, err := client.DescribeAddonVersions(ctx, &eks.DescribeAddonVersionsInput{AddonName: addonName, KubernetesVersion: k8sVersion})
	if err != nil {
		log.Println("ERROR:  Failure getting addon versions ", err)
	}
	// Find the default version.  TECHNICALLY this is a paginated call, so need to long term add support for that.
	var defaultVersion = ""
	for _, addon := range versions.Addons {
		for _, addonVersion := range addon.AddonVersions {
			for _, capability := range addonVersion.Compatibilities {
				if capability.DefaultVersion == true {
					defaultVersion = *addonVersion.AddonVersion
				}
			}
		}
	}
	if len(defaultVersion) == 0 {
		log.Println("WARN: Failed to find valid addon for " + *addonName + " ... skipping updates of the addon!")
		// Should we return err instead of exiting?
		return
	}
	//NOMINALLY we should check if there's a service account/config and apply that here not just default to node settinsg :)
	response, err := client.UpdateAddon(ctx, &eks.UpdateAddonInput{
		AddonName:        addonName,
		ClusterName:      clusterName,
		AddonVersion:     &defaultVersion,
		ResolveConflicts: "OVERWRITE",
	})
	log.Println("INFO: Updating addon " + *addonName + " to " + defaultVersion + ".   Id of update is:" + *response.Update.Id + " ... waiting for completion!")
	waiter := eks.NewAddonActiveWaiter(client)
	waitErr := waiter.Wait(ctx, &eks.DescribeAddonInput{
		AddonName:   addonName,
		ClusterName: clusterName,
	}, time.Duration(20)*time.Minute)
	if waitErr != nil {
		log.Println("ERROR: Failure on addon update in the time allowed!", waitErr)
	}
}

func updateClusterNodeGroup(client *eks.Client, ctx context.Context, clusterName *string, nodegroupName *string, waitForNodeUpdates int) error {
	version, err := client.UpdateNodegroupVersion(ctx, &eks.UpdateNodegroupVersionInput{ClusterName: clusterName, NodegroupName: nodegroupName})
	if err != nil {
		log.Println("ERROR: Update call failed", err)
		return err
	}
	log.Println("INFO: Upgrade job started... " + *version.Update.Id)
	waiter := eks.NewNodegroupActiveWaiter(client)
	waitErr := waiter.Wait(ctx, &eks.DescribeNodegroupInput{ClusterName: clusterName, NodegroupName: nodegroupName}, time.Duration(waitForNodeUpdates)*time.Minute)
	if waitErr != nil {
		log.Println("ERROR: Update failed to complete in the allotted time", err)
		return waitErr
	}
	return nil
}
