# List all Web Jump Items
data "sra_web_jump_list" "all" {}

# Filter by URL
data "sra_web_jump_list" "filtered" {
  url = "https://exciting.site"
}
