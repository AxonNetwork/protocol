
# AWS + Terraform notes

## Required environment variables

Terraform expects a `secrets.auto.tfvars` file in this folder.  It should contain the following values:

- `AWS_ACCESS_KEY`
- `AWS_SECRET_KEY`
- `AWS_REGION`
- `BUGSNAG_API_KEY`


## Updating ECS agent on an instance

```sh
$ aws ecs update-container-agent --cluster swarm-node --container-instance f9539123-27e3-4132-a3c2-1abb5e98a798
```

(Get the `--container-instance` ID from the ECS dashboard)

