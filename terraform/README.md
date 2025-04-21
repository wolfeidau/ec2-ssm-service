# terraform for SSM Parameter Store

Initialize Terraform:

```
terraform init
```

Then run:

```
terraform apply -var="buildkite_token=your-token" -var="honeycomb_api_key=your-key"
```
