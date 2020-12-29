#cloud-config
ssh_authorized_keys:
  - ${ssh_key}

groups:
  - docker

# Add default auto created user to docker group
system_info:
  default_user:
    groups: [docker]

package_update: true

packages:
  - git
  - docker.io
  - runc

runcmd:
  - curl -sLSf https://github.com/containerd/containerd/releases/download/v1.3.5/containerd-1.3.5-linux-amd64.tar.gz > /tmp/containerd.tar.gz && tar -xvf /tmp/containerd.tar.gz -C /usr/local/bin/ --strip-components=1
  - curl -SLfs https://raw.githubusercontent.com/containerd/containerd/v1.3.5/containerd.service | tee /etc/systemd/system/containerd.service
  - systemctl daemon-reload && systemctl start containerd
  - /sbin/sysctl -w net.ipv4.conf.all.forwarding=1
  - mkdir -p /opt/cni/bin
  - curl -sSL https://github.com/containernetworking/plugins/releases/download/v0.8.5/cni-plugins-linux-amd64-v0.8.5.tgz | tar -xz -C /opt/cni/bin
  - mkdir -p /go/src/github.com/openfaas/
  - mkdir -p /var/lib/faasd/secrets/
  - echo ${gw_password} > /var/lib/faasd/secrets/basic-auth-password
  - echo admin > /var/lib/faasd/secrets/basic-auth-user
  - cd /go/src/github.com/openfaas/ && git clone --depth 1 --branch 0.9.10 https://github.com/openfaas/faasd
  - curl -fSLs "https://github.com/openfaas/faasd/releases/download/0.9.10/faasd" --output "/usr/local/bin/faasd" && chmod a+x "/usr/local/bin/faasd"
  - cd /go/src/github.com/openfaas/faasd/ && /usr/local/bin/faasd install
  - systemctl status -l containerd --no-pager
  - journalctl -u faasd-provider --no-pager
  - systemctl status -l faasd-provider --no-pager
  - systemctl status -l faasd --no-pager
  - curl -sSLf https://cli.openfaas.com | sh
  - sleep 5 && journalctl -u faasd --no-pager
  - systemctl daemon-reload
  - sleep 60
  - mkdir -p /var/lib/spinner
  - cd /var/lib/spinner
  - git clone https://github.com/felipecruz91/spinner.git
  - cd spinner
  - echo ${gw_password} | faas-cli login -g http://localhost:8080 --password-stdin
  - echo -n ${hcloud_token} > secret-api-key.txt
  - faas-cli secret create secret-api-key --from-file=secret-api-key.txt -g http://localhost:8080
  - faas-cli secret list -g http://localhost:8080
  - sed -i s/\$FAASD_NODE_IP/localhost/g spinner.yml
  - sed -i s/\$FAASD_NODE_IP/localhost/g spinner-controller.yml
  - sed -i s/\$DOCKER_USER/${docker_user}/g spinner.yml
  - sed -i s/\$DOCKER_USER/${docker_user}/g spinner-controller.yml
  - faas-cli deploy -f spinner.yml
  - faas-cli deploy -f spinner-controller.yml
