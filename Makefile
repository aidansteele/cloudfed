default: deploy run

deploy:
	terraform init
	terraform apply
	terraform output -json > tfoutput.json

run:
	go run aws/aws.go
	go run azure/azure.go
	go run gcp/gcp.go
