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
	goreleaser build --single-target