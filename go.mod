module absa-demo

go 1.13

require (
	github.com/jaxxstorm/pulumi-rke/sdk/v2 v2.0.0-20200506200825-580758eddd4d
	github.com/pulumi/pulumi-aws/sdk/v2 v2.3.0
	github.com/pulumi/pulumi-kubernetes/sdk/v2 v2.0.0
	github.com/pulumi/pulumi/sdk/v2 v2.1.0
)

replace github.com/jaxxstorm/pulumi-rke/sdk/v2 => ../go/src/github.com/jaxxstorm/pulumi-rke/sdk
