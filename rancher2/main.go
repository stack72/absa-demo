package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v2/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v2/go/aws/rds"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v2/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v2/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v2/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v2/go/kubernetes/providers"
	"github.com/pulumi/pulumi-rancher2/sdk/v2/go/rancher2"
	"github.com/pulumi/pulumi-random/sdk/v2/go/random"
	"github.com/pulumi/pulumi/sdk/v2/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v2/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		// Set some configuration values
		config := config.New(ctx, "")
		awsAccessKey := config.RequireSecret("awsAccessKey")
		awsSecretKey := config.RequireSecret("awsSecretKey")

		// Create an EKS cluster using rancher2
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

		// Export our kubeconfig so we can use it locally
		ctx.Export("kubeconfig", cluster.KubeConfig)

		// Create a random password
		// Generate a random password.
		passwordArgs := random.RandomPasswordArgs{
			Length:  pulumi.Int(20),
			Special: pulumi.Bool(true),
		}
		password, err := random.NewRandomPassword(ctx, "password", &passwordArgs,
			pulumi.AdditionalSecretOutputs([]string{"result"}))
		if err != nil {
			return err
		}

		// Grab the VPC we created as part of the EKS creation
		vpc, err := ec2.LookupVpc(ctx, &ec2.LookupVpcArgs{
			Tags: map[string]interface{}{
				"displayName": "eks-8077e36",
			},
		})
		if err != nil {
			return err
		}

		// Get the subnet IDs from the VPC
		subnets, err := ec2.GetSubnetIds(ctx, &ec2.GetSubnetIdsArgs{VpcId: vpc.Id})
		if err != nil {
			return err
		}

		// Create a subnet group to use for our RDS database
		dbSubnetGroup, _ := rds.NewSubnetGroup(ctx, "wordpressSubnetGroup", &rds.SubnetGroupArgs{
			Name:      pulumi.String("wordpress-db"),
			SubnetIds: toPulumiStringArray(subnets.Ids),
		})

		// Create an RDS database
		db, err := rds.NewInstance(ctx, "db", &rds.InstanceArgs{
			Name:               pulumi.String("wordpress"),
			Engine:             pulumi.String("mysql"),
			EngineVersion:      pulumi.String("5.7"),
			InstanceClass:      pulumi.String("db.t2.micro"),
			ParameterGroupName: pulumi.String("default.mysql5.7"),
			AllocatedStorage:   pulumi.Int(5),
			Username:           pulumi.String("wordpress"),
			Password:           password.Result,
			DbSubnetGroupName:  dbSubnetGroup.Name,
		})

		// Create a k8s provider
		k8sProvider, err := providers.NewProvider(ctx, "k8sprovider", &providers.ProviderArgs{
			Kubeconfig: cluster.KubeConfig,
		}, pulumi.DependsOn([]pulumi.Resource{cluster}))
		if err != nil {
			return err
		}

		// Create a secret to store the wordpress database password
		secret, err := corev1.NewSecret(ctx, "wordpress-password", &corev1.SecretArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Labels: pulumi.StringMap{
					"app": pulumi.String("wordpress"),
				},
				Name: pulumi.String("wordpress-password"),
			},
			StringData: pulumi.StringMap{
				"wordpress-password": password.Result, // We need to encrypt this in state
			},
		}, pulumi.Provider(k8sProvider))

		_ = secret
		// Create the Wordpress deployment
		// Redis leader Deployment
		_, err = appsv1.NewDeployment(ctx, "wordpress", &appsv1.DeploymentArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Labels: pulumi.StringMap{
					"app": pulumi.String("wordpress"),
				},
			},
			Spec: appsv1.DeploymentSpecArgs{
				Selector: &metav1.LabelSelectorArgs{
					MatchLabels: pulumi.StringMap{
						"app": pulumi.String("wordpress"),
					},
				},
				Replicas: pulumi.Int(1),
				Template: &corev1.PodTemplateSpecArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Labels: pulumi.StringMap{
							"app": pulumi.String("wordpress"),
						},
					},
					Spec: &corev1.PodSpecArgs{
						Containers: corev1.ContainerArray{
							corev1.ContainerArgs{
								Name:  pulumi.String("wordpress"),
								Image: pulumi.String("wordpress:4.8-apache"),
								Ports: corev1.ContainerPortArray{
									&corev1.ContainerPortArgs{
										ContainerPort: pulumi.Int(80),
									},
								},
								Env: corev1.EnvVarArray{
									corev1.EnvVarArgs{
										Name:  pulumi.String("WORDPRESS_DB_HOST"),
										Value: db.Endpoint,
									},
									corev1.EnvVarArgs{
										Name: pulumi.String("WORDPRESS_DB_PASSWORD"),
										ValueFrom: corev1.EnvVarSourceArgs{
											SecretKeyRef: corev1.SecretKeySelectorArgs{
												Name: pulumi.String("wordpress-password"), // We need to calculate this using apply,
												Key:  pulumi.String("wordpress-password"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}, pulumi.Provider(k8sProvider))
		if err != nil {
			return err
		}

		return nil
	})
}

func toPulumiStringArray(a []string) pulumi.StringArrayInput {
	var res []pulumi.StringInput
	for _, s := range a {
		res = append(res, pulumi.String(s))
	}
	return pulumi.StringArray(res)
}
