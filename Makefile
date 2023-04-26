default: install generate

generate:
	go generate ./...

install: tidy
	go install .

tidy:
	go mod tidy

test:
	go test -count=1 -parallel=4 ./tests

testacc:
	TF_ACC=1 go test -count=1 -parallel=4 -timeout 10m -v ./tests

tfapply: install
	cd ./test-tf-files && terraform apply

tfplan: install
	cd ./test-tf-files && terraform plan

tfshow: install
	cd ./test-tf-files && terraform show

strelease:
	goreleaser build --single-target --snapshot --clean

testrelease:
	mkdir -p test-reg/registry.terraform.io/beyondtrust/beyondtrust-sra/1.0.0/darwin_amd64
	goreleaser build --single-target --snapshot --clean --output ./test-reg/registry.terraform.io/beyondtrust/beyondtrust-sra/1.0.0/darwin_amd64/terraform-provider-beyondtrust-sra_v1.0.0