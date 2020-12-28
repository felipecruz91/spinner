resource "hcloud_ssh_key" "default" {
  name       = "my-ssh-key"
  public_key = file("~/.ssh/id_rsa.pub")
}

data "local_file" "ssh_key" {
  filename = pathexpand(var.ssh_key_file)
}

resource "random_password" "password" {
  length           = 16
  special          = true
  override_special = "_-#"
}

data "template_file" "cloud_init" {
  template = file("cloud-config.tpl")
  vars = {
    gw_password = random_password.password.result,
    ssh_key     = data.local_file.ssh_key.content
  }
}

resource "hcloud_server" "faasd_node" {
  name        = var.faasd_node_name
  image       = "ubuntu-20.04"
  server_type = var.faasd_node_server_type
  location    = var.faasd_node_location
  ssh_keys    = [hcloud_ssh_key.default.name]
  user_data   = data.template_file.cloud_init.rendered
}
