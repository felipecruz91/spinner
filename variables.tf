variable "hcloud_token" {
  default     = ""
  description = "Hetzner Cloud API token"
}

variable "faasd_node_name" {
  default     = "faasd-node"
  description = "Name of the faasd_node node"
}

variable "faasd_node_server_type" {
  default     = "cx11"
  description = "Server type of the faasd_node node"
}

variable "faasd_node_location" {
  default     = "nbg1"
  description = "Location of the faasd_node node"
}

variable "ssh_key_file" {
  default     = "~/.ssh/id_rsa.pub"
  description = "Path to the SSH public key file"
}
