# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

**This file is maintained by hand.** Add your entry in the same pull request as
the change — see [CONTRIBUTING.md](CONTRIBUTING.md#changelog). It was previously
regenerated from commit messages by `git-chglog`, which overwrote hand-written
entries on every release and dropped any commit using a scope (`chore(deps):`),
so almost nothing was captured. That automation has been removed.

<a name="Unreleased"></a>
## [Unreleased]

### Fixed

- `sra_vault_ssh_account` / `sra_vault_token_account` / `sra_vault_username_password_account`: a `jump_item_association` with `filter_type = "any_jump_items"` or `"no_jump_items"` can now be created. With no nested `criteria` block the provider sent `"criteria": null`, which the appliance rejects with `422 This value must be an array.`, so neither of those two `filter_type` values worked at all — only `"criteria"` did. The field is now omitted when it is not set; omitting it is accepted, sending null is not.
- `sra_vault_ssh_account` / `sra_vault_token_account` / `sra_vault_username_password_account` / `sra_vault_account_group`: a `jump_item_association` with `filter_type = "criteria"` but nothing to filter on is now rejected at plan time instead of failing part-way through an apply. The API requires `criteria` or `jump_items` when the filter type is `"criteria"`, and separately rejects a `criteria` block whose properties are all empty; neither precondition was expressed in the schema. The error names the attribute and the alternatives.
- `sra_vault_ssh_account`: setting `private_key_public_cert` to an empty string no longer fails the apply. From PRA 25.1 the appliance rejects both `""` and `null` for this field, so a configuration that passes an unset variable through to it — a common module pattern — could not be applied at all. The provider now omits the field when no certificate is set, keeps the configured value across a refresh, and no longer marks the attribute `Computed`, since the API does not return it on a read.
- The provider now recovers when the appliance stops accepting its API token, instead of failing part-way through an apply. The appliance retains a bounded number of concurrent tokens per set of credentials and evicts the oldest, so a long-running or concurrent workflow — several `terraform` commands under one service account, or parallel workspaces — could have its token invalidated while still in use. That surfaced as a bare `status: 401` diagnostic with nothing to act on, after resources had already been created. The client now re-authenticates once and replays the request; if that also fails the original error is returned, so genuinely bad credentials still fail fast rather than looping.
- Creating an API client no longer requests two tokens where one is needed, which halves this provider's contribution to the limit described above.
- Out-of-band deletions are now detected: a resource deleted outside Terraform is recreated on the next apply instead of failing the plan with a `404`.
- `sra_jump_client_installer`: `elevate_install` / `elevate_prompt` no longer flip to `false` after apply (the create response does not echo them back).
- `sra_network_tunnel_jump`: the provider no longer crashes when `filter_rules` is null, empty, or malformed.
- `sra_jump_group` / `sra_jumpoint`: removing all `group_policy_memberships` now applies cleanly instead of erroring with an inconsistent-result; group policy membership refresh now works and detects drift.
- `sra_vault_account_group`: errors reading the `jump_item_association` sub-resource are no longer silently swallowed; a read failure now raises a diagnostic instead of leaving stale state with no signal.
- `sra_vault_ssh_account` / `sra_vault_token_account` / `sra_vault_username_password_account`: an absent `jump_item_association` is now recorded as absent — on an ordinary refresh, and when a read fails transiently. The provider previously wrote an empty association into state, a `filter_type` of `""` that no configuration can produce and the attribute does not accept, and because nothing downstream read that as absence, three things went wrong. An account that never had an association showed a difference on every plan, which applying could not settle. Removing a `jump_item_association` block applied cleanly but left the association in state, so the next apply tried to delete it a second time and errored. An association deleted outside Terraform — the sub-resource, not the account itself — was not recreated; the apply failed with `Account does not have an Asset association.` State written by an earlier version converges on the first plan after upgrading, with no apply required.
- `sra_vault_account_group`: Update issued a `POST` to the `jump_item_association` sub-resource whenever state held no association — exactly the case left by a fresh `terraform import` — but the endpoint documents only `GET`/`PATCH`, so the apply failed. Update now always `PATCH`es.
- Group policy membership refresh no longer reports a confusing "cannot unmarshal object" error when the API response is genuinely malformed; the real decode error is now surfaced.
- Fixed a goroutine and read-lock leak on every group policy membership operation that returned an error mid-loop.
- API layer hardening: removed unsafe pointer usage and panics from the model transforms (no more provider crashes on unexpected types), moved product state onto the client (concurrency-safe), and checked previously-ignored ID-parse errors.
- Provider documentation rendered `\"BT_API_HOST\"` with literal backslashes, and `web_jump`'s `username_format` lost its list of accepted values whenever docs were regenerated. Both are fixed at the schema, so `go generate ./...` is now lossless (the index page's frontmatter summary renders as flat text under tfplugindocs 0.24; the page body is unaffected).
- `sra_postgresql_tunnel_jump`: documentation was published under a filename that did not resolve on the Terraform Registry.
- `sra_protocol_tunnel_jump`: corrected a `useranme` typo in the usage example.

### Changed

- `sra_vault_ssh_account` / `sra_vault_token_account` / `sra_vault_username_password_account`: `jump_item_association` is no longer `Computed`. It was, which told Terraform the provider would supply a value when the configuration did not — so for an account whose configuration declares no `jump_item_association` block, a plan that was in fact **removing** an association rendered it as `(known after apply)` under `1 to change`. An association attached outside Terraform, or carried in by `terraform import`, was deleted by the next apply that changed any other attribute, with nothing in the plan saying so. Since an association scopes where a stored credential may be injected, that is a change worth seeing. The removal is now shown in full, naming the criteria being dropped.

  Two consequences to expect. An association that exists on the appliance but not in your configuration now shows a difference until you either declare it or let it be removed — previously that difference was hidden, not absent. And `sra_vault_account_group` is unchanged: it carries a schema default, genuinely does supply a value the configuration omits, and keeps `Computed`.

- Provider debug logging no longer includes API request or response bodies. Log lines now record the method, URL, endpoint and payload size instead of the payload itself, and the plan/state dumps on create, read, update and delete no longer print the resource's attribute values. Debug output is substantially smaller, and no longer contains the values of the attributes being managed. If you were relying on `TF_LOG=DEBUG` to inspect exact request payloads, use the appliance's own API logs.
- Failed `create` diagnostics no longer echo the request body back in the error message. The error text now carries only the API's own error; diagnostics are shown to the operator regardless of `TF_LOG`.
- `group_policy_id` is now validated as a numeric ID on every resource that accepts it (`sra_jump_group`, `sra_jumpoint`, `sra_vault_account_group`, `sra_vault_ssh_account`, `sra_vault_token_account`, `sra_vault_username_password_account`). This matches the Configuration API, which types the field as `integer, minimum: 1`. A configuration supplying a non-numeric value now fails at plan time with a clear message rather than building a malformed request path. Values sourced from `data.sra_group_policy_list` — the documented pattern — are unaffected.
- Raised the minimum Go version needed to build the provider from source to 1.26.0 (previously 1.23.7 with a 1.24.1 toolchain pin).

### Known issues

- `terraform import` of `sra_jump_client_installer` forces a destroy/recreate on the next plan, and this cannot be fixed provider-side. `elevate_install`, `elevate_prompt` and `valid_duration` are never refreshed from the API, so imported state holds no value for them and the next plan sees a difference on attributes that require replacement. Recreating an installer invalidates any copies already distributed.

  Verified against a live appliance on 2026-09-15: an installer created with `elevate_install: true` and `elevate_prompt: true` is returned as `false` for both by the `POST` **and** by a subsequent `GET /jump-client/installer/{id}`, and `valid_duration` is absent from the read response entirely. Because the API never reports the real values, there is no read the provider could trust; the fields are deliberately excluded from refresh instead. Import these resources only if you are prepared for the first apply to replace them.

- `terraform import` of `sra_vault_account_group` does not round-trip: the first plan after import always shows a difference on `jump_item_association` and `group_policy_memberships`, and the first apply rewrites them.

  Both attributes are read back from the API on refresh, but the provider only writes them into state when the pre-refresh value is already non-null (`readJIA` at `bt/rs/vault_account_group.go:263`, and the equivalent early return in `ReadGPMemberships`). After `terraform import`, state holds only `id`, so both stay null while the configuration declares them — and `jump_item_association` additionally carries a non-null schema default. The import itself succeeds and the subsequent apply is not destructive; it PATCHes the association into place.

  Not fixed because the guard is not import-specific: removing it changes refresh behaviour for every existing account group, not just imported ones. Tracked for a future release.

### Dependencies

- Bump terraform-plugin-framework to 1.19.0, terraform-plugin-framework-validators to 0.19.0, terraform-plugin-log to 0.10.0, and terraform-plugin-go to 0.31.0.
- Bump terratest to 1.0.x, terraform-plugin-docs to 0.24.x, pgx to 5.9.x, deckarep/golang-set to 2.9.x and spdystream to 0.5.1.
- Promote `terraform-plugin-go` to a direct dependency (used by the new helper unit tests).

### Internal

- Extracted shared generic Group Policy membership and Jump Item Association CRUD helpers (removing ~1,200 lines of duplicated resource code) and genericized `DiffGPLists`; deleted dead code and replaced `golang.org/x/exp/slices` with the stdlib.
- Pinned the CI build/lint/E2E Go toolchain to `go.mod` (fixes the `go >= 1.26` build failures) and excluded the E2E `test/` directory from `golangci-lint`.
- Resolved all `golangci-lint` findings and added unit tests for the model transforms, `DiffGPLists`, and the Group Policy membership / Jump Item Association helpers.
- Removed the `git-chglog` release workflow and its `.chglog/` config; this file is now maintained by hand.

<a name="v1.3.0"></a>
## [v1.3.0] - 2025-09-15

> Reconstructed after the fact. This release shipped without a changelog entry,
> and its contents were mistakenly listed under `Unreleased` until now.

### Added

- 25.2 API support: updated models and resources for new / changed Jump / Tunnel types (PostgreSQL / MySQL / Network / Protocol) and Jump Client Installer adjustments.

### Fixed

- Compatibility fixes for network tunnel and jump client installer resources against 25.2 API changes.

### Dependencies

- Bump terraform-plugin-framework to 1.15.x and validators to 0.18.x, terraform-plugin-docs to 0.22.x, and terratest to 0.50.x.
- Dependency updates: oauth2, net, crypto, circl, xz, deckarep/golang-set, testify and others.

### Internal

- Added the Semgrep workflow and pinned GitHub Action SHAs for supply-chain security.
- Narrowed CODEOWNERS.
- GitHub Actions updates: checkout, download-artifact, upload-pages-artifact, upload-artifact, setup-go, goreleaser-action, golangci-lint-action, codeql-action, create-pull-request, ghaction-import-gpg.

---

<a name="v1.2.0"></a>
## [v1.2.0] - 2024-06-24

- Changes for SRA 24.2 releases


<a name="v1.1.0"></a>
## [v1.1.0] - 2024-03-04

### Feat
- Changes for SRA 24.1 releases
  - Includes new "Token" Vault account type

### Chore
- fix a couple of typos in the description

### Pull Requests
- Merge pull request [#102](https://github.com/beyondtrust/terraform-provider-sra/issues/102) from BeyondTrust/chore/update-to-fix-depbot-changes
- Merge pull request [#101](https://github.com/beyondtrust/terraform-provider-sra/issues/101) from BeyondTrust/dependabot/github_actions/actions/download-artifact-4.1.4
- Merge pull request [#99](https://github.com/beyondtrust/terraform-provider-sra/issues/99) from BeyondTrust/dependabot/go_modules/golang.org/x/oauth2-0.17.0
- Merge pull request [#100](https://github.com/beyondtrust/terraform-provider-sra/issues/100) from BeyondTrust/chore/24.1-updates
- Merge pull request [#98](https://github.com/beyondtrust/terraform-provider-sra/issues/98) from BeyondTrust/dependabot/go_modules/github.com/hashicorp/terraform-plugin-framework-1.6.0
- Merge pull request [#96](https://github.com/beyondtrust/terraform-provider-sra/issues/96) from BeyondTrust/dependabot/github_actions/actions/upload-artifact-4.3.1
- Merge pull request [#95](https://github.com/beyondtrust/terraform-provider-sra/issues/95) from BeyondTrust/dependabot/github_actions/autero1/action-terraform-3.0.1
- Merge pull request [#94](https://github.com/beyondtrust/terraform-provider-sra/issues/94) from BeyondTrust/dependabot/github_actions/golangci/golangci-lint-action-4
- Merge pull request [#93](https://github.com/beyondtrust/terraform-provider-sra/issues/93) from BeyondTrust/update-changelog


<a name="v1.0.6"></a>
## [v1.0.6] - 2024-02-05
### Pull Requests
- Merge pull request [#89](https://github.com/beyondtrust/terraform-provider-sra/issues/89) from BeyondTrust/dependabot/go_modules/github.com/hashicorp/terraform-plugin-docs-0.18.0
- Merge pull request [#86](https://github.com/beyondtrust/terraform-provider-sra/issues/86) from BeyondTrust/dependabot/github_actions/actions/deploy-pages-4
- Merge pull request [#91](https://github.com/beyondtrust/terraform-provider-sra/issues/91) from BeyondTrust/dependabot/github_actions/actions/download-artifact-4.1.2
- Merge pull request [#88](https://github.com/beyondtrust/terraform-provider-sra/issues/88) from BeyondTrust/dependabot/go_modules/golang.org/x/oauth2-0.16.0
- Merge pull request [#90](https://github.com/beyondtrust/terraform-provider-sra/issues/90) from BeyondTrust/dependabot/go_modules/github.com/gruntwork-io/terratest-0.46.11
- Merge pull request [#87](https://github.com/beyondtrust/terraform-provider-sra/issues/87) from BeyondTrust/dependabot/go_modules/github.com/hashicorp/terraform-plugin-framework-1.5.0
- Merge pull request [#84](https://github.com/beyondtrust/terraform-provider-sra/issues/84) from BeyondTrust/dependabot/github_actions/peter-evans/create-pull-request-6
- Merge pull request [#83](https://github.com/beyondtrust/terraform-provider-sra/issues/83) from BeyondTrust/dependabot/github_actions/actions/upload-artifact-4.3.0
- Merge pull request [#82](https://github.com/beyondtrust/terraform-provider-sra/issues/82) from BeyondTrust/update-changelog


<a name="v1.0.5"></a>
## [v1.0.5] - 2024-01-25
### Fix
- workflow dispatch test trigger

### Pull Requests
- Merge pull request [#73](https://github.com/beyondtrust/terraform-provider-sra/issues/73) from BeyondTrust/dependabot/go_modules/github.com/deckarep/golang-set/v2-2.6.0
- Merge pull request [#74](https://github.com/beyondtrust/terraform-provider-sra/issues/74) from BeyondTrust/dependabot/github_actions/actions/configure-pages-4
- Merge pull request [#75](https://github.com/beyondtrust/terraform-provider-sra/issues/75) from BeyondTrust/dependabot/github_actions/actions/upload-pages-artifact-3
- Merge pull request [#76](https://github.com/beyondtrust/terraform-provider-sra/issues/76) from BeyondTrust/dependabot/github_actions/crazy-max/ghaction-import-gpg-6.1.0
- Merge pull request [#80](https://github.com/beyondtrust/terraform-provider-sra/issues/80) from BeyondTrust/dependabot/go_modules/github.com/gruntwork-io/terratest-0.46.9
- Merge pull request [#77](https://github.com/beyondtrust/terraform-provider-sra/issues/77) from BeyondTrust/dependabot/github_actions/actions/upload-artifact-4.0.0
- Merge pull request [#78](https://github.com/beyondtrust/terraform-provider-sra/issues/78) from BeyondTrust/dependabot/github_actions/actions/download-artifact-4.1.0
- Merge pull request [#79](https://github.com/beyondtrust/terraform-provider-sra/issues/79) from BeyondTrust/dependabot/go_modules/github.com/cloudflare/circl-1.3.7
- Merge pull request [#71](https://github.com/beyondtrust/terraform-provider-sra/issues/71) from BeyondTrust/dependabot/go_modules/golang.org/x/crypto-0.17.0
- Merge pull request [#68](https://github.com/beyondtrust/terraform-provider-sra/issues/68) from BeyondTrust/dependabot/go_modules/github.com/deckarep/golang-set/v2-2.5.0
- Merge pull request [#69](https://github.com/beyondtrust/terraform-provider-sra/issues/69) from BeyondTrust/dependabot/go_modules/golang.org/x/oauth2-0.15.0
- Merge pull request [#70](https://github.com/beyondtrust/terraform-provider-sra/issues/70) from BeyondTrust/dependabot/go_modules/github.com/gruntwork-io/terratest-0.46.7
- Merge pull request [#67](https://github.com/beyondtrust/terraform-provider-sra/issues/67) from BeyondTrust/dependabot/go_modules/github.com/gruntwork-io/terratest-0.46.6
- Merge pull request [#65](https://github.com/beyondtrust/terraform-provider-sra/issues/65) from BeyondTrust/dependabot/go_modules/golang.org/x/oauth2-0.13.0
- Merge pull request [#66](https://github.com/beyondtrust/terraform-provider-sra/issues/66) from BeyondTrust/dependabot/go_modules/github.com/hashicorp/terraform-plugin-framework-1.4.2
- Merge pull request [#63](https://github.com/beyondtrust/terraform-provider-sra/issues/63) from BeyondTrust/dependabot/go_modules/google.golang.org/grpc-1.57.1
- Merge pull request [#62](https://github.com/beyondtrust/terraform-provider-sra/issues/62) from BeyondTrust/dependabot/go_modules/golang.org/x/net-0.17.0


<a name="v1.0.4"></a>
## [v1.0.4] - 2023-10-05
### Chore
- update API reference for 23.3.1

### Feat
- updates for 23.3.1 compatibility

### Pull Requests
- Merge pull request [#61](https://github.com/beyondtrust/terraform-provider-sra/issues/61) from BeyondTrust/update-changelog
- Merge pull request [#60](https://github.com/beyondtrust/terraform-provider-sra/issues/60) from BeyondTrust/23.3.1-changes
- Merge pull request [#59](https://github.com/beyondtrust/terraform-provider-sra/issues/59) from BeyondTrust/dependabot/go_modules/github.com/gruntwork-io/terratest-0.44.1
- Merge pull request [#51](https://github.com/beyondtrust/terraform-provider-sra/issues/51) from BeyondTrust/dependabot/go_modules/github.com/hashicorp/terraform-plugin-framework-1.4.0
- Merge pull request [#52](https://github.com/beyondtrust/terraform-provider-sra/issues/52) from BeyondTrust/dependabot/go_modules/golang.org/x/oauth2-0.12.0
- Merge pull request [#53](https://github.com/beyondtrust/terraform-provider-sra/issues/53) from BeyondTrust/dependabot/go_modules/github.com/gruntwork-io/terratest-0.44.0
- Merge pull request [#54](https://github.com/beyondtrust/terraform-provider-sra/issues/54) from BeyondTrust/dependabot/go_modules/github.com/hashicorp/terraform-plugin-framework-validators-0.12.0
- Merge pull request [#55](https://github.com/beyondtrust/terraform-provider-sra/issues/55) from BeyondTrust/dependabot/github_actions/goreleaser/goreleaser-action-5.0.0
- Merge pull request [#56](https://github.com/beyondtrust/terraform-provider-sra/issues/56) from BeyondTrust/dependabot/github_actions/actions/upload-artifact-3.1.3
- Merge pull request [#57](https://github.com/beyondtrust/terraform-provider-sra/issues/57) from BeyondTrust/dependabot/github_actions/crazy-max/ghaction-import-gpg-6.0.0
- Merge pull request [#58](https://github.com/beyondtrust/terraform-provider-sra/issues/58) from BeyondTrust/dependabot/github_actions/actions/checkout-4
- Merge pull request [#49](https://github.com/beyondtrust/terraform-provider-sra/issues/49) from BeyondTrust/dependabot/go_modules/github.com/deckarep/golang-set/v2-2.3.1
- Merge pull request [#47](https://github.com/beyondtrust/terraform-provider-sra/issues/47) from BeyondTrust/dependabot/go_modules/golang.org/x/oauth2-0.11.0
- Merge pull request [#48](https://github.com/beyondtrust/terraform-provider-sra/issues/48) from BeyondTrust/dependabot/go_modules/github.com/hashicorp/terraform-plugin-framework-validators-0.11.0
- Merge pull request [#46](https://github.com/beyondtrust/terraform-provider-sra/issues/46) from BeyondTrust/dependabot/go_modules/github.com/hashicorp/terraform-plugin-framework-1.3.4
- Merge pull request [#45](https://github.com/beyondtrust/terraform-provider-sra/issues/45) from BeyondTrust/dependabot/go_modules/github.com/gruntwork-io/terratest-0.43.12
- Merge pull request [#44](https://github.com/beyondtrust/terraform-provider-sra/issues/44) from BeyondTrust/dependabot/github_actions/goreleaser/goreleaser-action-4.4.0
- Merge pull request [#50](https://github.com/beyondtrust/terraform-provider-sra/issues/50) from BeyondTrust/fix-workflow-ref
- Merge pull request [#43](https://github.com/beyondtrust/terraform-provider-sra/issues/43) from BeyondTrust/update-changelog
- Merge pull request [#42](https://github.com/beyondtrust/terraform-provider-sra/issues/42) from BeyondTrust/add-changelog-workflow


<a name="v1.0.3"></a>
## [v1.0.3] - 2023-08-08
### Chore
- Try to work around dependabot's inability to read ENV values

### Pull Requests
- Merge pull request [#39](https://github.com/beyondtrust/terraform-provider-sra/issues/39) from BeyondTrust/doc-updates
- Merge pull request [#41](https://github.com/beyondtrust/terraform-provider-sra/issues/41) from BeyondTrust/fix-terratest-terraform-env
- Merge pull request [#40](https://github.com/beyondtrust/terraform-provider-sra/issues/40) from BeyondTrust/tweak-config-and-workflows
- Merge pull request [#37](https://github.com/beyondtrust/terraform-provider-sra/issues/37) from BeyondTrust/dependabot/go_modules/github.com/hashicorp/terraform-plugin-docs-0.16.0
- Merge pull request [#36](https://github.com/beyondtrust/terraform-provider-sra/issues/36) from BeyondTrust/dependabot/go_modules/github.com/hashicorp/terraform-plugin-framework-1.3.3
- Merge pull request [#35](https://github.com/beyondtrust/terraform-provider-sra/issues/35) from BeyondTrust/dependabot/go_modules/github.com/gruntwork-io/terratest-0.43.11
- Merge pull request [#38](https://github.com/beyondtrust/terraform-provider-sra/issues/38) from BeyondTrust/work-around-dependabot-secrets-restrictions
- Merge pull request [#34](https://github.com/beyondtrust/terraform-provider-sra/issues/34) from BeyondTrust/dependabot/go_modules/golang.org/x/oauth2-0.10.0
- Merge pull request [#33](https://github.com/beyondtrust/terraform-provider-sra/issues/33) from BeyondTrust/dependabot/github_actions/actions/upload-pages-artifact-2
- Merge pull request [#32](https://github.com/beyondtrust/terraform-provider-sra/issues/32) from BeyondTrust/dependabot/github_actions/goreleaser/goreleaser-action-4.3.0
- Merge pull request [#31](https://github.com/beyondtrust/terraform-provider-sra/issues/31) from BeyondTrust/dependabot/go_modules/golang.org/x/oauth2-0.9.0
- Merge pull request [#30](https://github.com/beyondtrust/terraform-provider-sra/issues/30) from BeyondTrust/dependabot/go_modules/github.com/hashicorp/terraform-plugin-docs-0.15.0
- Merge pull request [#29](https://github.com/beyondtrust/terraform-provider-sra/issues/29) from BeyondTrust/dependabot/go_modules/github.com/gruntwork-io/terratest-0.43.6
- Merge pull request [#28](https://github.com/beyondtrust/terraform-provider-sra/issues/28) from BeyondTrust/dependabot/go_modules/github.com/hashicorp/terraform-plugin-framework-1.3.2


<a name="v1.0.2"></a>
## [v1.0.2] - 2023-07-05

[Unreleased]: https://github.com/beyondtrust/terraform-provider-sra/compare/v1.3.0...HEAD
[v1.3.0]: https://github.com/beyondtrust/terraform-provider-sra/compare/v1.2.0...v1.3.0
[v1.2.0]: https://github.com/beyondtrust/terraform-provider-sra/compare/v1.1.0...v1.2.0
[v1.1.0]: https://github.com/beyondtrust/terraform-provider-sra/compare/v1.0.6...v1.1.0
[v1.0.6]: https://github.com/beyondtrust/terraform-provider-sra/compare/v1.0.5...v1.0.6
[v1.0.5]: https://github.com/beyondtrust/terraform-provider-sra/compare/v1.0.4...v1.0.5
[v1.0.4]: https://github.com/beyondtrust/terraform-provider-sra/compare/v1.0.3...v1.0.4
[v1.0.3]: https://github.com/beyondtrust/terraform-provider-sra/compare/v1.0.2...v1.0.3
[v1.0.2]: https://github.com/beyondtrust/terraform-provider-sra/compare/v1.0.1...v1.0.2
