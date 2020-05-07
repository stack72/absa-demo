# Set up a rancher cluster

We need some environment variables

```bash
export RANCHER_URL="xxxxxxxx"
export RANCHER_ACCESS_KEY="xxxxxxxxx"
export RANCHER_SECRET_KEY="xxxxxxxxxxxxxxxxx"
export RANCHER_INSECURE="true"
```

Then we need to set a couple vars:

```bash
pulumi config set --secret awsAccessKey
pulumi config set --secret awsSecretKey QQWEWFRGEFBERHTEBWBWEBWV
```
