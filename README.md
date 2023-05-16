<a href="https://www.beyondtrust.com">
    <img src=".github/beyondtrust_logo.svg" alt="BeyondTrust" title="BeyondTrust" align="right" height="50">
</a>

# BeyondTrust SRA Terraform Provider

The [BeyondTrust SRA Provider](https://registry.terraform.io/providers/beyondtrust/sra/latest/docs) allows [Terraform](https://terraform.io) to manage access to resources in SRA using SRA's configuration API.

See the SRA Provider documentation as well as the Configuration API documentation in your instance for more information on supported endpoints and parameters.

As of the initial release, this provider requires Remote Support or Privileged Remote Access version 23.2.1+. Using this provider with prior versions could result in Terraform reporting errors that cannot be addressed.

## Configuration

To function, the provider requires the hostname of your appliance as well as credentials for an API account. This API account must have permission to access the Configuration API. If you also plan to manage Vault accounts with Terraform, then the API account also needs the "Manage Vault Accounts" permission.

```terraform
provider "sra" {
  host          = "example.beyondtrust.com"
  client_id     = "api account client id"
  client_secret = "api account client secret"
}
```

These values can also be passed by setting the environment variables `BT_API_HOST`, `BT_CLIENT_ID`, and `BT_CLIENT_SECRET`, which are the same environment settings used by the `btapi` CLI tool.

## Getting Help

For assistance or to report any issues, please contact BeyondTrust technical support.
