package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jaxxstorm/pulumi-rke/sdk/v2/go/rke"
	"github.com/pulumi/pulumi-aws/sdk/v2/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v2/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v2/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v2/go/pulumi/config"

	//appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v2/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v2/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v2/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v2/go/kubernetes/providers"
)

type RkeNode struct {
	InternalAddress pulumi.StringOutput
	Address         pulumi.StringOutput
	Role            []string
	User            string
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		config := config.New(ctx, "")

		sshKeyName := config.Get("sshKeyname")
		sshPubKey := config.Get("sshPubKey")

		// Get the defaults if the config isn't set
		if sshKeyName == "" {
			sshKeyName = "absa-demo"
		}
		if sshPubKey == "" {
			sshPubKey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDN+yEA4ST7AByox+TqYqXDxwm+c+Ggnc1SoA7ueGhSPyq2wBWLKcvy3dPwIsSeLyYYwk4CRtJwPakesZctnYIIUzwO4t5s6KUuGft+QkCf/THFvNglAq6yRBFU5p0aeH9p5jfvBIBZgCNQDUsTm4vk6h1Q6iNzyCc0LJZoWdrHDJXGl414Zr1IKQEpFcQ94ovSmNaAA7Dz3ibileSPbMqw9qS5OWvZw+cs+i1Dh7G2ln6AmIJpejVcDnQn9zycqh5P2U8gwLX66bSuQ2A326epY231Eq5LTD9rPDj5f2uxys5SSjs8sXV0ohpCs4nWdlKg72pVuzRki4l/z/iMfk/MK3f0P0QGxFzCFNEc7UvgH4T4p7tFbX2Rxb7jZdtshalR86tX1JWFNGuvJUFDnQF0cb3x+1lJXvpFhTEJ1d//LjzDlH/WftIh5rqnUIJdWj05dshzV+E5C0bofXCEQCMpPJVx+Lg5UxHwi6u/5rwF6d7a6mRkx5iG0yC5KvP6BMxmpRF9xQPa3rzpd/Dd63g9RmoL702sP7j345ipp54t8SRmdJ5zI30gaktUNHxetWZy78hWDT57U+pL/AZMn1xJmCktB+0UrDfULaF/JYcwuB6Ih2mVFwaXkIQoDnvX2W+BFr8blU2F5pD4XHcKs2/97LlWTaHwndrA0upILnaTpQ== stack72@demo-mbp.local"
		}

		// Get the ID for the latest Amazon Linux AMI.
		mostRecent := true
		ami, err := aws.GetAmi(ctx, &aws.GetAmiArgs{
			Filters: []aws.GetAmiFilter{
				{
					Name:   "name",
					Values: []string{"*amzn2*ecs*"},
				},
			},
			Owners:     []string{"amazon"},
			MostRecent: &mostRecent,
		})
		if err != nil {
			return err
		}

		// Create an SSH KeyPair to be used with our ec2 instances
		key, err := ec2.NewKeyPair(ctx, "demo-keypair", &ec2.KeyPairArgs{
			KeyName:   pulumi.String(sshKeyName),
			PublicKey: pulumi.String(sshPubKey),
		})

		/*
		* RKE will SSH into the nodes and then run various checks on the created instances
		* We open this port, as well as some ports for the k8s API and running some healtchecks
		 */
		group, err := ec2.NewSecurityGroup(ctx, "rke-secgrp", &ec2.SecurityGroupArgs{
			Ingress: ec2.SecurityGroupIngressArray{
				ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(22),
					ToPort:     pulumi.Int(22),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
				ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(6443),
					ToPort:     pulumi.Int(6443),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
				ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(2379),
					ToPort:     pulumi.Int(2380),
					Self:       pulumi.Bool(true),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				ec2.SecurityGroupEgressArgs{
					Protocol:   pulumi.String("-1"),
					FromPort:   pulumi.Int(0),
					ToPort:     pulumi.Int(0),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
		})
		if err != nil {
			return err
		}

		var nodes []RkeNode
		// We need 3 nodes for our control plane, so create a loop and make a new
		// EC2 instance
		for x := 0; x <= 2; x++ {
			srv, err := ec2.NewInstance(ctx, fmt.Sprintf("demo-server-%d", x), &ec2.InstanceArgs{
				InstanceType:        pulumi.String("m5.large"),
				VpcSecurityGroupIds: pulumi.StringArray{group.ID()},
				Ami:                 pulumi.String(ami.Id),
				KeyName:             key.KeyName,
				UserData: pulumi.String(`#!/bin/bash
docker pull rancher/hyperkube:v1.17.4-rancher1`),
			})
			if err != nil {
				return err
			}

			// Append some details from our EC2 instances to our RKE node struct
			// This is then used to build our RKE cluster
			nodes = append(nodes, RkeNode{
				Address:         srv.PublicIp.ToStringOutput(),
				InternalAddress: srv.PrivateIp.ToStringOutput(),
				User:            "ec2-user",
				Role: []string{
					"controlplane",
					"etcd",
				},
			})
		}

		// This is a hack and must not be used in production
		// this ensures the instances are alive in AWS!!
		time.Sleep(time.Minute * 3)

		// Set up our SSH key path which is on our machine
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		sshKeyPath := filepath.Join(cwd, "rsa")

		// Create a new RKE cluster!
		cluster, err := rke.NewCluster(ctx, "demo-cluster", &rke.ClusterArgs{
			ClusterName:         pulumi.String("absa-cluster"),
			IgnoreDockerVersion: pulumi.Bool(false),
			Network: &rke.ClusterNetworkArgs{
				Plugin: pulumi.String("canal"),
			},
			SshKeyPath: pulumi.String(sshKeyPath),
			Nodes:      buildNodeDetails(nodes),
			Authentication: &rke.ClusterAuthenticationArgs{
				Sans: getNodeAddresses(nodes),
			},
		})
		if err != nil {
			return err
		}

		// Exports the cluster YAML from the RKE cluster
		// and sets it as a provider to some Kubernetes resources
		k8sProvider, err := providers.NewProvider(ctx, "k8sprovider", &providers.ProviderArgs{
			Kubeconfig: cluster.KubeConfigYaml,
		}, pulumi.DependsOn([]pulumi.Resource{cluster}))
		if err != nil {
			return err
		}

		// Create a new namespace for rancher
		_, err = corev1.NewNamespace(ctx, "cattle-system", &corev1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("cattle-system"),
			},
		}, pulumi.Provider(k8sProvider))

		// Create a new namespace for nginx-ingress
		_, err = corev1.NewNamespace(ctx, "nginx", &corev1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("nginx-ingress"),
			},
		}, pulumi.Provider(k8sProvider))

		// Export our kubeconfig so we can use it locally
		ctx.Export("kubeconfig", cluster.KubeConfigYaml)

		return nil
	})
}

func buildNodeDetails(nodes []RkeNode) rke.ClusterNodeArray {
	var result rke.ClusterNodeArray
	for _, node := range nodes {
		result = append(result, rke.ClusterNodeArgs{
			Address: node.Address,
			User:    pulumi.String(node.User),
			Roles:   getNodeRoles(node),
		})
	}

	return result
}

func getNodeAddresses(nodes []RkeNode) pulumi.StringArrayInput {
	var res []pulumi.StringInput
	for _, node := range nodes {
		res = append(res, node.InternalAddress)
	}
	return pulumi.StringArray(res)
}

func getNodeRoles(nodes RkeNode) pulumi.StringArrayInput {
	var res []pulumi.StringInput
	for _, role := range nodes.Role {
		res = append(res, pulumi.String(role))
	}
	return pulumi.StringArray(res)
}
