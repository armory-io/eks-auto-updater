package aws

import (
	"context"
	"log"
	"time"

	"github.com/armory-io/eks-auto-updater/pkg/aws/eks"
	"github.com/armory-io/eks-auto-updater/pkg/aws/options"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type Client interface {
	EKS() eks.Interface
}

type client struct {
	eks  eks.Interface
	opts options.Options
}

func NewClient(ctx context.Context, opts ...options.Option) (Client, error) {
	c := &client{}

	for _, o := range opts {
		o.Apply(&c.opts)
	}
	AWSRegion := c.opts.AWSRegion
	AWSRoleArn := c.opts.AWSRoleArn

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(AWSRegion))
	if err != nil {
		return nil, err
	}

	if len(AWSRoleArn) != 0 {
		stsClient := sts.NewFromConfig(cfg)
		provider := stscreds.NewAssumeRoleProvider(stsClient, AWSRoleArn, func(o *stscreds.AssumeRoleOptions) {
			o.RoleSessionName = "eks-auto-updater"
			o.Duration = time.Duration(60) * time.Minute
		})
		cfg.Credentials = aws.NewCredentialsCache(provider)

		log.Println("INFO: Assuming role ARN " + AWSRoleArn)
	}

	if c.eks, err = eks.NewFromConfig(cfg); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *client) EKS() eks.Interface {
	return c.eks
}
