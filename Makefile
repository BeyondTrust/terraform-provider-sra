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

# -parallel 4, not Go's default of GOMAXPROCS: every test holds API tokens and
# writes memberships on the same group policy. -timeout 30m matches CI.
teste2e: testrelease
	go test -v -timeout 30m -parallel 4 ./test

# Lists (dry-run, the default) or deletes leaked E2E fixture objects. See
# CONTRIBUTING.md before setting E2E_SWEEP=delete. Never sets
# E2E_SWEEP_CONFIRM_HOST: type that yourself.
sweepe2e:
	E2E_SWEEP=$${E2E_SWEEP:-dry-run} go test -count=1 -v -timeout 60m -run '^TestE2EOrphanSweep$$' ./test

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
