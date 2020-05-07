# Questions

## Why Pulumi and not Terraform?

- Full programming language support, not a DSL _MAIN POINT_!
- More robust execution engine
- Secret encryption in Statefile

## Why not write Kubernetes controllers/operators for everything?

- Controllers/operators will manage the applicaion lifecycle
- However, controllers/operators still have configuration complexity across multiple clusters
  - For example, a controller will have a CRD which needs to be on all your clusters. We support this natively, tf does a horrible job here
  - You'll also need to install the controller deployment, Pulumi can do this for you
  - Pulumi + operators work hand in hand to give you the best cloud native experience

## How does Pulumi fit in a GitOps world

- With GitOps is that you only need a YAML file
  - You need to install a gitops controller like ArgoCD or Flux - Pulumi is great here!
  - You also need to register that application with your gitops controller, generally with a CRD. Again, Pulumi can register your application remotely
  - Pulumi will also render YAML files to disk, so you can use Pulumi's languages to remove the configuration complexity, and let your gitops controller manage the deployment

## Benefits of SaaS? Maybe show console, centralised policies, RBAC etc.


## Talk about being able to reuse most Terraform providers
