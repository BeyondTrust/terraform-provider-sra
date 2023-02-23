# BT SRA Terraform Provider

 Readme (and code) written by someone who doesn't really know Go for others who also probably don't know Go†

 _† If you do know Go, feel free to submit PRs with corrections 😀_

 ## Purpose
 This terraform provider exists to allow customers to manage access to resources they are managing with terraform in SRA. The idea being that they can define the SRA things (jump items, jump groups, whatever) in the same place they are defining the actual resources in AWS or Azure or wherever. With that scope in mind, this is not intended to be a general interface to the entire config API—rather the parts of the API that might be helpful when provisioning assets at scale using terraform

 ## Code Structure
 Go defines a package per folder, so organizing code into a logical directory structure can be a bit of a challenge if the pieces all need to be aware of each other. We have done our best to make that happen.

 * /api: this is the API client where the code that actually talks to the appliance lives
   * client.go: the main APIClient info. Exports the APIClient interface which holds al the other methods
   * crud.go: genericized top level methods that will allow you to query the expected API operations for a given model… the model provided controls what is actually being called. these are top level functions because if they are tied to a specific receiver (that is, a member function on the APIClient), then the receiver has to include the generic type, not the function itself. the client did not need to be per-model generic
   * models.go: model files representing the various API endpoints that we will query. model files should have all properties defined using proper Go types. values that are truly optional (that is, can be present and null in a response) should be pointers. json tags should be listed for each field for proper json mapping. each model must implement the id() and endpoint() methods to meet the requirement to use the generic crud functions.
* /bt: here live all the terraform stuff
  * /ds: directory for all of the datasource related things
    * Things that need to live here TBD
  * /models: model files that are mapped to terraform types, and have tags mapping to what we expect the users to provide in their terraform files; these mappings should mirror the json mappings. **IMPORTANT** the fields and names must match exactly to the API models for the same resources. the terraform code has a lot of boilerplate, and I tried to make this generic to avoid having to do it everywhere. in order to do that, we have to be able to generically copy back and forth between terraform types and json types, and to do that the fields need to be identical, and the types mapped correctly (e.g. string maps to types.String)
  * /rs: the resources related things. things that need crud operations fall into this category