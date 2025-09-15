# TODO — Jump Items & Vault (OpenAPI vs provider)

Purpose
- Capture the comparison between `openapi/*-configuration.openapi.yaml` (PRA/RS) and the provider implementation under `bt/` for jump items and vault features so we don't forget gaps or follow-ups.

Checklist (what this file covers)
- [ ] Record API fields/endpoints present in OpenAPI but missing or incomplete in `bt/` (jump items + vault)
- [ ] Prioritize fixes and map to files
- [ ] Provide recommended next actions (A/B/C) and verification commands

Summary of findings (concise)
- Provider implements most common jump-item types (Remote RDP, Remote VNC, Shell, Web, ProtocolTunnel, JumpClientInstaller) and many vault features (VaultAccount list datasource, VaultAccountGroup resource, group-policy mappings).
- Notable gaps found:
  - Jump Client Installer: missing support-button fields and some installer options supported by the API.
  - Vault: some API account types (e.g., `VaultAwsSecretAccount`, `VaultPasswordSafeAccount`) are not represented in provider models or datasource validators.
  - There is no obvious `vault_account` resource implementation for creating/updating/deleting all supported account types (verify whether intentionally omitted).
- Previously resolved issues (kept for history):
  - `quality` enum for Remote RDP required `best_performance` (was added to `bt/rs/remote_rdp.go`).
  - Timestamp JSON parsing was updated in `api/json.go` to accept RFC3339 string timestamps as well as numeric seconds.

Concrete items (priority order)

High priority
1) Add JumpClientInstaller support-button and installer options
   - API fields (from `openapi/bt-rs-configuration.openapi.yaml` `/jump-client/installer` request schema):
     - `support_button_profile_code_name` (string)
     - `allow_override_support_button_profile` (boolean)
     - `support_button_direct_queue` (string)
     - `allow_override_support_button_direct_queue` (boolean)
     - `valid_duration` constraints/defaults (integer)
   - Files to change:
     - `bt/models/jump_items.go` (add fields to `JumpClientInstaller` struct)
     - The resource that creates installers (verify location in `bt/rs/` — add fields to schema)
   - Rationale: exposes installer customization available in API.

2) Vault account type coverage
   - Add models and schema support for API account types not present in provider:
     - `VaultAwsSecretAccount`
     - `VaultPasswordSafeAccount`
   - Files to change:
     - `bt/models/vault.go` (add structs)
     - `bt/ds/vault_account.go` (expand `type` validator list)
     - Add or update resource(s) if `vault_account` creation is intended to be supported.
   - Rationale: API supports these types; provider currently limits types returned/created.

Medium priority
3) Ensure `vault_account` resource exists (create if missing)
   - Verify whether provider intends to allow create/delete of Vault accounts. If yes, implement `bt/rs/vault_account.go` to POST appropriate `oneOf` request bodies for supported types.

4) Verify JumpClientInstaller RS-specific fields are fully exposed
   - Fields to confirm/expose: `is_quiet`, `customer_client_start_mode`, `attended_session_policy_id`, `unattended_session_policy_id`, `allow_override_attended_session_policy`, `allow_override_unattended_session_policy`.
   - Files: `bt/models/jump_items.go` and installer resource schema.

Low priority / informational
- API provides copy endpoints (`/jump-item/*/copy`) and vault checkout/checkin/rotate/force-check-in endpoints. These are not typical Terraform resources; document them but do not implement as resources unless required by workflow.

Suggested next actions (pick one)
- A) Implement JumpClientInstaller support-button fields and default/validation handling (quick win). Estimated: small (1–2 files).
- B) Implement `VaultAwsSecretAccount` / `VaultPasswordSafeAccount` models and expand datasource validators (medium effort).
- C) Implement `vault_account` resource for create/update/delete of supported account types (larger effort).

Verification / test commands
```bash
cd /Volumes/Code/go/terraform-provider-sra
# run specific tests after implementing changes
go test ./test -run TestJumpClientInstaller -v
go test ./test -run TestJumpointAndJumpGroup -v
go test ./test -run TestRemoteRDP -v
```

Notes & references
- OpenAPI specs:
  - `openapi/bt-rs-configuration.openapi.yaml`
  - `openapi/bt-pra-configuration.openapi.yaml`
- Provider locations:
  - jump models: `bt/models/jump_items.go`
  - jump resources: `bt/rs/remote_rdp.go`, `bt/rs/remote_vnc.go`, `bt/rs/shell_jump.go`, `bt/rs/web_jump.go`, etc.
  - installer model: `bt/models/jump_items.go` (JumpClientInstaller)
  - vault datasource: `bt/ds/vault_account.go`
  - vault account group resource: `bt/rs/vault_account_group.go`

History (do not delete)
- Timestamp unmarshal fix: `api/json.go` updated to accept RFC3339 strings.
- Remote RDP `best_performance` enum added.

Owner / next owner
- Whoever picks this up: update the TODO with PR/commit references when done.


---

If you want, I can start with task A (add support-button fields to installer) and create the required code changes + tests — tell me to proceed and I will implement them now.
