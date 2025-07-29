See the corresponding blog post: https://awsteele.com/blog/2025/07/27/federating-into-azure-gcp-and-aws-with-oidc.html

1. Comment out any of the modules in `main.tf` that you don't want to use (e.g.
AWS, GCP, Azure). 
2. Fill out the value sin `vars.auto.tfvars`. 
3. Run `make`. This will deploy the Terraform and save its output in a format 
   that the Go apps can read. It will then run each Go app.
