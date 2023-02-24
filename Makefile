default: install

generate:
	go generate ./...

install:
	go install .

test:
	go test -count=1 -parallel=4 ./tests

testacc:
	TF_ACC=1 go test -count=1 -parallel=4 -timeout 10m -v ./tests

tfapply: install
	cd ./examples/install-verification && terraform apply

tfplan: install
	cd ./examples/install-verification && terraform plan

tfshow: install
	cd ./examples/install-verification && terraform show