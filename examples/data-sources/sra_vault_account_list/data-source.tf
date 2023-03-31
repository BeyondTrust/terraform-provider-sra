# List all Web Jump Items
data "sra_web_jump_list" "all" {}

# Filter by tag
data "sra_web_jump_list" "filtered" {
  url = "https://exciting.site"
}
