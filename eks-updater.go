package main

import (
	"context"
	"flag"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"log"
	"os"
	"time"
)

func main() {
	// Load the Shared AWS Configuration (~/.aws/config)
	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
		os.Exit(5)
	}

	// Create an Amazon S3 service client
	client := eks.NewFromConfig(cfg)

	clusterName := validateOrExit("cluster-name", "", "Cluster name REQUIRED")
	nodegroupName := validateOrExit("nodegroup-name", "", "Cluster name REQUIRED")
	waitForNodeUpdates := *flag.Int("nodegroup-wait-time", 120, "Time in minutes to wait for node group update to complete.  Defaults to 120 minutes")
	updateCluster(client, ctx, clusterName, nodegroupName, waitForNodeUpdates)
	updateAddon(client, ctx, clusterName, nodegroupName, strref("kube-proxy"), strref("1.22"))

}

func strref(s string) *string {
	return &s
}

func updateAddon(client *eks.Client, ctx context.Context, clusterName *string, nodegroupName *string, addonName *string, k8sVersion *string) {
	versions, err := client.DescribeAddonVersions(ctx, &eks.DescribeAddonVersionsInput{AddonName: addonName, KubernetesVersion: k8sVersion})
	if err != nil {
		log.Fatal(err)
		os.Exit(5)
	}

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
		log.Fatal("Failed to find valid addon for " + *addonName)
		os.Exit(2)
	}

}

func updateCluster(client *eks.Client, ctx context.Context, clusterName *string, nodegroupName *string, waitForNodeUpdates int) {
	version, err := client.UpdateNodegroupVersion(ctx, &eks.UpdateNodegroupVersionInput{ClusterName: clusterName, NodegroupName: nodegroupName})
	if err != nil {
		log.Fatal("Update call failed", err)
		return
	}
	log.Println("Upgrade job started... " + *version.Update.Id)
	waiter := eks.NewNodegroupActiveWaiter(client)
	waiter.Wait(ctx, &eks.DescribeNodegroupInput{ClusterName: clusterName, NodegroupName: nodegroupName}, time.Duration(waitForNodeUpdates)*time.Minute)
}

func validateOrExit(parameterName string, defaultValue string, description string) *string {

	value := flag.String(parameterName, defaultValue, description)
	if (len(*value)) == 0 {
		log.Fatal("Invalid value for " + parameterName + " Must be set!")
		os.Exit(1)
	}
	return value
}
