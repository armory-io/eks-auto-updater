package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	_ "github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/hashicorp/go-version"
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
	roleArn := flag.String("role-arn", "", "Role to assume if set")
	region := flag.String("region", "us-west-2", "Region to operate in - defaults to us-west-2")
	waitTimeForNodeUpdates := *flag.Int("nodegroup-wait-time", 120, "Time in minutes to wait for node group update to complete.  Defaults to 120 minutes")
	addonsToUpdate := strings.Split(*flag.String("addons-to-update", "kube-proxy,coredns,vpc-cni,aws-ebs-csi-driver", "Comma separated list of adds on to updates.  Defaults to kube-proxy, coredns, vpc-cni, aws-ebs-csi-driver addons"), ",")
	flag.Parse()
	if len(*clusterName) == 0 {
		log.Fatal("Invalid cluster name!  Must be set!")
	}

	// Load the Shared AWS Configuration (~/.aws/config)
	ctx := context.TODO()
	client, err := getEksClient(ctx, *region, *roleArn)
	if err != nil {
		log.Fatal("Unable to get EKS client:", err)
	}

	log.Println("INFO: Starting updates...")
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
		err = updateClusterNodeGroup(client, ctx, clusterName, nodegroupName, waitTimeForNodeUpdates)
		if err != nil {
			log.Fatal("ERROR: Unable to update cluster node group... ", err)
		}
	}
	for _, addon := range addonsToUpdate {
		err = updateAddon(client, ctx, clusterName, &addon, clusterInformation.Cluster.Version)
		if err != nil {
			log.Fatal("Unable to update addon "+addon, err)
		}

	}
	log.Println("INFO:  Updates complete!")

}

func updateAddon(client *eks.Client, ctx context.Context, clusterName *string, addonName *string, k8sVersion *string) (err error) {

	addonInfo, err := client.DescribeAddon(ctx, &eks.DescribeAddonInput{AddonName: addonName, ClusterName: clusterName})
	if err != nil {
		log.Println("WARN:  Error describing addon " + *addonName + " in the cluster... is it actually in the cluster? Skipping...")
		return nil
	}

	versions, err := client.DescribeAddonVersions(ctx, &eks.DescribeAddonVersionsInput{AddonName: addonName, KubernetesVersion: k8sVersion})
	if err != nil {
		log.Println("ERROR:  Failure getting addon versions ", err)
		return err
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
		log.Println("WARN: Failed to find valid addon for " + *addonName + " ... skipping updates of the addon!")
		// Should we return err instead of exiting?
		return nil
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
	//NOMINALLY we should check if there's a service account/config and apply that here not just default to node settinsg :)
	response, err := client.UpdateAddon(ctx, &eks.UpdateAddonInput{
		AddonName:        addonName,
		ClusterName:      clusterName,
		AddonVersion:     &defaultVersion,
		ResolveConflicts: "OVERWRITE",
	})
	if err != nil {
		return err
	}
	log.Println("INFO: Addon update triggered ... ID of  " + *response.Update.Id + "... waiting for completion")
	err = eks.NewAddonActiveWaiter(client).Wait(ctx, &eks.DescribeAddonInput{
		AddonName:   addonName,
		ClusterName: clusterName,
	}, time.Duration(20)*time.Minute)
	return err
}

func updateClusterNodeGroup(client *eks.Client, ctx context.Context, clusterName *string, nodegroupName *string, waitForNodeUpdates int) (err error) {
	version, err := client.UpdateNodegroupVersion(ctx, &eks.UpdateNodegroupVersionInput{ClusterName: clusterName, NodegroupName: nodegroupName})
	if err != nil {
		return fmt.Errorf("ERROR: Update call failed %w", err)
	}
	log.Println("INFO: Upgrade job started... " + *version.Update.Id)
	waiter := eks.NewNodegroupActiveWaiter(client)
	err = waiter.Wait(ctx, &eks.DescribeNodegroupInput{ClusterName: clusterName, NodegroupName: nodegroupName}, time.Duration(waitForNodeUpdates)*time.Minute)
	if err != nil {
		return fmt.Errorf("ERROR: Update failed to complete in the allotted time: %w", err)
	}
	return nil
}

func getEksClient(ctx context.Context, region string, roleArn string) (client *eks.Client, err error) {

	var cfg aws.Config
	cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(region))

	if err != nil {
		return client, err
	}
	if len(roleArn) == 0 {
		return eks.NewFromConfig(cfg), err
	}
	log.Println("INFO: Assuming role ARN " + roleArn)
	// Create config & sts client with source account

	sourceAccount := sts.NewFromConfig(cfg)
	// Default and only support 1 hour duration.  We MAY hit an issue here particularly if node groups take a LONG time to update.
	duration := int32(3600)
	// Assume target role and store credentials
	rand.Seed(time.Now().UnixNano())
	response, err := sourceAccount.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         aws.String(roleArn),
		RoleSessionName: aws.String("eks-auto-updater-" + strconv.Itoa(10000+rand.Intn(25000))),
		DurationSeconds: &duration,
	})
	if err != nil {
		return client, err
	}
	var assumedRoleCreds = response.Credentials

	// Create config with target service client, using assumed role
	cfg, err = config.LoadDefaultConfig(ctx, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(*assumedRoleCreds.AccessKeyId, *assumedRoleCreds.SecretAccessKey, *assumedRoleCreds.SessionToken)), config.WithRegion(region))
	if err != nil {
		return client, err
	}
	return eks.NewFromConfig(cfg), err
}
