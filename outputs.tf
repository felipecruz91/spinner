output "gateway_url" {
  value = "http://${hcloud_server.faasd_node.ipv4_address}:8080"
}

output "password" {
  value = random_password.password.result
}

output "login_cmd" {
  value = "faas-cli login -g http://${hcloud_server.faasd_node.ipv4_address}:8080 -p ${random_password.password.result}"
}
