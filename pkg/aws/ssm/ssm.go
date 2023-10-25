package ssm

import (
	"context"

	"github.com/armory-io/eks-auto-updater/pkg/aws/eks"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type Interface interface {
	GetLatestAMIReleaseVersion(ctx context.Context, version, clusterName *string) (string, error)
}

type Client struct {
	ssm       *ssm.Client
	eksClient eks.Interface
}

func NewFromConfig(cfg aws.Config, eksClient eks.Interface) (Interface, error) {
	c := &Client{}

	c.ssm = ssm.NewFromConfig(cfg)
	c.eksClient = eksClient

	return c, nil
}

func (c Client) GetLatestAMIReleaseVersion(ctx context.Context, version, clusterName *string) (string, error) {
	parameter, err := c.ssm.GetParameter(ctx, &ssm.GetParameterInput{
		Name: aws.String("/aws/service/eks/optimized-ami/" + *version + "/amazon-linux-2/recommended/release_version"),
	})
	if err != nil {
		return "", err
	}

	return *parameter.Parameter.Value, nil
}
