# Contributing to the BeyondTrust SRA Terraform provider

Thank you for your interest in contributing to our project!

Here is some information on how to get started and where to ask for help.

## Getting Started

The SRA Terraform provider is a translation layer between Terraform and the SRA Configuration API. Thus, the documentation for the configuration API directly applies to the Terraform resources defined. All Terraform schemas should directly map to a configuration API endpoint. Not all endpoints in the configuration API have Terraform counterparts.

The configuration API documentation can be viewed from the /login interface of your instance under **Management > Security**.  You must be logged in with administrator permissions to reach this section of the interface.

## How can I Contribute?

### Reporting Bugs

Bugs should be submitted through BeyondTrust Support. Any bugs should be submitted against either _Remote Support_ or _Privileged Remote Access_, depending on which product you're using. Our support team will ensure the escalation is raised to the proper team internally.

If the bug is a security vulnerability, instead please refer to the [responsible disclosure section of our security policy](https://www.beyondtrust.com/security#disclosure).

### Feature Requests

Feature requests should also be submitted through BeyondTrust Support, also against either _Remote Support_ or _Privileged Remote Access_, depending on which product you're using. Submitting through our support organization will ensure the request gets send to the proper Product Management team for consideration.

### Making Changes and Submitting a Pull Request

#### **Did you write a patch that fixes a bug?**

- Open a GitHub pull request with the patch.
- Ensure the PR description clearly describes both the problem and the solution. If you have a support ticket, please include that number as well.
- We will review the changes and make a determination if we will accept the change or not.

#### **Do you intend to add a new feature or change an existing one?**

- Consider submitting a feature request through BeyondTrust Support to ensure that your proposed changes do not conflict with new features that are already planned or in development.
- If you do open a PR, please ensure the description clearly describes what the change is, and what problem your change is solving.
- Any new code must include unit tests (if possible) or end-to-end tests (if Terraform resources are changed or added). All tests must pass.
- We will review the change and determine if it fits within our goals for the project.

### Tests

Please note that all tests must pass for any change submitted to be accepted. This includes both the unit tests within the modules as well as the end-to-end tests found under `./test`.

#### Running End-to-End tests

This project includes end-to-end tests for all data source and resource types defined by the provider defined in `./test`. These test are written using the [Terratest](https://terratest.gruntwork.io) framework. To run them, you will need to have valid configuration for the SRA provider in your process environment.

**The tests will connect to the configured appliance and create/destroy infrastructure in SRA during the tests.**

**We do not recommend running these tests on a production appliance.**

The E2E tests can be easily run with the `Makefile` shortcut, which builds the current snapshot into a release Terratest can use:

```sh
make teste2e
```

#### Running Unit Tests

Unit tests can be run manually with the `go test` command by excluding the `./test` directory:

```sh
go build -v ./...
go test $(go list ./... | grep -v /test)
```

or by using the `Makefile` shortcut:

```sh
make unittest
```
