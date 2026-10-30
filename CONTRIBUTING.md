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

### Changelog

`CHANGELOG.md` is maintained by hand. If your change is visible to someone using
the provider, add an entry under `## [Unreleased]` **in the same pull request**.

Use the [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) headings:

| Heading | Use it for |
| --- | --- |
| `Added` | New resources, data sources, or attributes |
| `Changed` | Behavior that differs from the previous release |
| `Deprecated` | Attributes or resources scheduled for removal |
| `Removed` | Anything deleted in this release |
| `Fixed` | Bug fixes |
| `Security` | Fixes for a vulnerability |
| `Known issues` | Shipping defects a user could hit, and how they present |
| `Dependencies` | Dependency bumps a user might care about |
| `Internal` | Refactors, CI, and tests with no user-visible effect |

Write for someone reading release notes, not for a reviewer reading the diff.
Name the resource, say what changed, and say what the user will now see:

> - `sra_jump_group`: removing all `group_policy_memberships` now applies cleanly
>   instead of failing with an inconsistent-result error.

Purely internal work (a refactor, a test, a lint fix) belongs under `Internal`
or can be left out entirely.

This file used to be regenerated from commit messages by `git-chglog`. That
automation was removed: it overwrote hand-written entries on every release, and
its commit pattern silently dropped anything with a scope, so `chore(deps):`
and similar never appeared. Please do not reintroduce generation without also
solving the overwrite problem — a per-pull-request entry directory, as the
upstream Terraform providers use, is the usual answer.

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

#### Cleaning up after interrupted E2E runs

A run that dies between `terraform apply` and its deferred `terraform destroy` (a cancelled CI job, a killed local run, a `go test` timeout) leaves its objects on the appliance. `TestE2EOrphanSweep` finds them and, when told to, deletes them. It does nothing unless `E2E_SWEEP` is set, and it reads every `E2E_SWEEP*` variable from the real process environment only: a `.env` file can't turn it on.

```sh
make sweepe2e                       # dry-run: list only
E2E_SWEEP=gp-snapshot make sweepe2e # read-only membership snapshot
```

| Variable | Meaning |
| --- | --- |
| `E2E_SWEEP` | `dry-run`, `delete` or `gp-snapshot`. Anything else fails the test. |
| `E2E_SWEEP_CONFIRM_HOST` | Delete mode only. Must equal `BT_API_HOST` (scheme, trailing slash and case are ignored). Type it yourself; never derive it from `$BT_API_HOST`. |
| `E2E_SWEEP_MIN_AGE` | Minimum marker age, as a Go duration. Default `24h`; delete mode never goes below `1h`. |
| `E2E_SWEEP_MAX` | Most objects of one type a delete may remove (default 20). Over it, a dry-run warns and a delete aborts before deleting anything. |
| `E2E_SWEEP_CANDIDATES_OUT` | Dry-run only. Writes the legacy candidates to this file for review. |
| `E2E_SWEEP_LEGACY_IDS` | Delete mode only. A reviewed candidates file; only the objects it lists are eligible among legacy ones. |

`gp-snapshot` lists the members of the first group policy, the one the fixtures add memberships to, as ids classed `fixture` or `non-fixture`. Compare a snapshot from before a run with one from after it: the non-fixture members should be identical, and no fixture members should remain.

**Fixture markers.** Every fixture object carries its run's marker in a name, tag or code name: `e2e`, then the creation time as 6 base36 digits of Unix minutes, then 6 random characters, for example `e2e0wq3k1a8x2pm`. Some objects carry it with a `2` appended. The sweeper matches only full fixture names (`This is a Name <marker> GP`, `Test JG <marker>`, and so on); for Jump Items, the name, the tag and a fixture Jump Group must all match. Group policies, users, jump policies, session policies, jump item roles, deployed Jump Clients and the Default account group are never deleted. Its output names fixture candidates only; everything else appears as a count, or as a type and id in a reason. The per-type counts cover fixture candidates (matched, eligible, legacy, excluded by reason), never every object of a type. A list that fails is reported as its status code or failure class, never its response body or URL.

**Delete mode needs an opt-in on the appliance.** Create a Jump Group by hand, once per test appliance, with code name `e2e_sweep_allowed` and comments set to the appliance's host in normalised form (no scheme, no trailing slash, lower case, for example `pra.example.com`). Without it, or if its comments name another host (a restored backup or a cloned appliance), delete mode refuses to run. Never create it on a production appliance.

```sh
read -r host </dev/tty   # type the host yourself
E2E_SWEEP=delete E2E_SWEEP_CONFIRM_HOST="$host" make sweepe2e
```

Before deleting, each object is read again and re-matched, and anything that contains or uses it is checked again on the appliance. An account group that holds other accounts, an account policy that something else uses (personal accounts included), and a Jump Group or Jumpoint that another item, installer, Jump Client or jump item association criteria uses are all kept. A Jumpoint is also kept when an installer or Jump Client uses it as its Jump Zone Proxy (`jumpoint_proxy`, the Jumpoint's name); if the list doesn't return that field, every Jumpoint is kept (`proxy-push refs unchecked`). Any error in one of those checks keeps the object.

For jump item association criteria, the sweeper reads the association of every account group and of every account in the Default group. It doesn't read the associations of accounts in other groups, because those accounts use their group's association. Some objects having no association (a 404) is normal; when every read for a type is a 404, the sweeper takes it for a wrong path and keeps every Jump Group (`jump-item-association-check-failed`). On an appliance where no Default-group account has an association, that keeps every Jump Group; giving one of those accounts an association clears it.

**Age.** A marker is eligible once it is older than `E2E_SWEEP_MIN_AGE`. The argument that it can't belong to a live run rests on CI's `go test -timeout 30m`. A local run with `-timeout 0`, or one paused in a debugger, can outlive the floor, and clock skew between machines shifts ages. Markers more than 5 minutes in the future are skipped.

**Legacy markers.** Runs before timestamped markers used 6 random characters, which carry no creation time. Those objects are never deleted on a pattern match alone:

1. Run a dry-run with `E2E_SWEEP_CANDIDATES_OUT=<file>`. The file starts with `# listed-at <time>` and has one `type<TAB>id<TAB># name` line per legacy candidate.
2. Review it, and delete the lines for anything you want to keep.
3. At least 35 minutes after the dry-run, run delete mode with `E2E_SWEEP_LEGACY_IDS=<file>`. Only the listed objects are deleted, each re-matched first. Any other line format is rejected.

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
