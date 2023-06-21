default: install generate

generate:
	go generate ./...

install: tidy
	go install .

tidy:
	go mod tidy

build: tidy
	go build -v ./...

unittest: build
	go test -v $$(go list ./... | grep -v test | xargs)

teste2e: testrelease
	go test -v -timeout 10m ./test

tfapply: install
	cd ./test-tf-files && terraform apply

tfplan: install
	cd ./test-tf-files && terraform plan

tfshow: install
	cd ./test-tf-files && terraform show

strelease:
	goreleaser build --single-target --snapshot --clean

testrelease:
	@DIR="./test-reg/registry.terraform.io/beyondtrust/sra/1.0.0/`go env GOOS`_`go env GOARCH`"; \
	rm -rf "./test-reg"; \
	mkdir -p $${DIR}; \
	goreleaser build --single-target --snapshot --clean --output $${DIR}/terraform-provider-sra_v1.0.0
