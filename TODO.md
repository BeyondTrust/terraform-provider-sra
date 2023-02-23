# TODO

## Resources
Things that are managed by TF (need CRUD)

### Jump Item resources
Only includes jump items via jumpoint
  * Shell Jump (DONE)
  * RDP
  * Web Jump
  * Protocol Tunnel
  * VNC

### Vault Resources
  * Accounts
  * Account Groups

### Other Resources
  * Jump Groups

### Jump Clients and Jumpoints
Figure out file providing on the endpoint

## Data sources
Things TF needs to query but doesn't manage

  * Need to abstract the DS classes like the resources
  * Implement filtering so you can query for a specific item
  * Will likely overlap with resources somewhat

DS Needed
  * Group Policy
  * … non jump item resources? (so you can query existing things)