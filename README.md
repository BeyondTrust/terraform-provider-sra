<a href="https://www.beyondtrust.com">
    <img src=".github/beyondtrust_logo.svg" alt="BeyondTrust" title="BeyondTrust" align="right" height="50">
</a>

# BeyondTrust SRA Terraform Provider

The [BeyondTrust SRA Provider](https://registry.terraform.io/providers/beyondtrust/sra/latest/docs) allows [Terraform](https://terraform.io) to manage access to resources in the [Secure Remote Access (SRA)](https://www.beyondtrust.com/secure-remote-access) products from BeyondTrust.  This module can be used with either the Remote Support and Privileged Remote Access products to interact with the Configuration API using appropriately configured API credentials.

See the SRA Provider documentation as well as the Configuration API documentation in your instance for more information on supported API endpoints and parameters.

This provider requires Remote Support or Privileged Remote Access version 23.2.1+. Using this provider with prior versions is not supported by BeyondTrust and could result in Terraform reporting errors.

## Use Cases

The chief use case for this provider is to manage access to all assets managed within your Terraform instance in conjection with BeyondTrust Remote Support or BeyondTrust Priviliged Remote Access products.

As examples, this provider allows:
* Enabling Jump Item creation and deletion to match the provisioning and deprovisioning of assets within Terraform.
* Enabling Vault credential creation and deletion to match the credentials used within the assets within Terraform.
* Enabling Vault credential associations to Jump Items to enable passwordless authentication to assets.
* Enabling Vault credential policy management to control how credentials are handled and used.
* Enabling Jump Group creation, deletion, and asset membership to leverage existing SRA access controls.
* Enabling Group Policy associations to Jump Groups, Vault Accounts, Vault Account Groups to control overall user access to all Terraform assets.

Examples for all of these use cases can be found within the [test-tf-files](https://github.com/BeyondTrust/terraform-provider-sra/tree/main/test-tf-files) section of our Github repo.

## Configuration

To function, the provider requires the hostname of your instance as well as credentials for an API account configured in that instance. This API account must have permission to "Allow Access" to the Configuration API. If you also plan to access or manage Vault accounts with Terraform, then the API account also needs the "Manage Vault Accounts" permission.

To use the API Account within your Terraform scripts, the hostname, Client ID, and Client Secret values should be passed by setting the environment "BT_API_HOST", "BT_CLIENT_ID", and "BT_CLIENT_SECRET" environment variables which are the same environment settings used by the btapi CLI tool.  While not recommended, it is also possible to set the values within the script itself with the following block.

```terraform
provider "sra" {
  host          = "<The SRA instance hostname, such as mycompanyname.beyondtrustcloud.com>"
  client_id     = "<The SRA API Account OAuth Client ID>"
  client_secret = "<The SRA API Account Client Secret>"
}
```



## Getting Help

For assistance or to report any issues, please contact [BeyondTrust Technical Support](https://www.beyondtrust.com/docs/index.htm#support)
