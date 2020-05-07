package main

import (
	"github.com/pulumi/pulumi-rancher2/sdk/v2/go/rancher2"
	"github.com/pulumi/pulumi/sdk/v2/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v2/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		config := config.New(ctx, "")

		awsAccessKey := config.RequireSecret("awsAccessKey")
		awsSecretKey := config.RequireSecret("awsSecretKey")

		cluster, err := rancher2.NewCluster(ctx, "eks", &rancher2.ClusterArgs{
			EksConfig: &rancher2.ClusterEksConfigArgs{
				AccessKey:         pulumi.Sprintf("%s", awsAccessKey),
				SecretKey:         pulumi.Sprintf("%s", awsSecretKey),
				Region:            pulumi.String("us-west-2"),
				KubernetesVersion: pulumi.String("1.15"),
			},
		})

		if err != nil {
			return err
		}

		_ = cluster

		return nil
	})
}
