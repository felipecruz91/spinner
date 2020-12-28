# spinner ⚡🔄

## Introduction

Serverless function built with [OpenFaaS](https://www.openfaas.com/) to spin new servers on Hetzner Cloud.

[![asciicast](https://asciinema.org/a/381485.svg)](https://asciinema.org/a/381485)

The serverless function is deployed on the cheapest VPS on Hetzner Cloud (running `faasd`).

## Use-cases

- Scaling nodes horizontally for video-encoding based on incoming http requests.

## Benefits

- Saving costs for large and expensive servers in Hetzner Cloud provider.

## Infrastructure provisioning

As of today, the cheapest server type on Hetzner Cloud is `cx11` (1vCPU, 2GB RAM for 3EUR/month). Learn more about the wide range of server types [here](https://www.hetzner.com/cloud).

Let's create a `cx11` server with `faasd` installed. We'll be using [cloud-init](https://cloudinit.readthedocs.io/en/latest/) to initialize the server with `faasd`, as per the [cloud-config.tpl](cloud-config.tpl) file.

```cli
git clone https://github.com/felipecruz91/spinner.git
cd spinner
```

To create the server on Hetzner Cloud, you need to have an Hetzner API token. Remember to set it in [main.tfvars](main.tfvars).

```
hcloud_token = "<YOUR_HCLOUD_TOKEN>"
```

Initialize Terraform and deploy the infrastructure:

```cli
terraform init
terraform apply -var-file=main.tfvars -auto-approve
```

If everything went well, you should see the following output:

```
gateway_url = http://<faasd-node-ip>:8080
login_cmd = faas-cli login -g http://<faasd-node-ip>:8080 -p <random-password>
password = <random-password>
```

The server should have been created. Let's export the following env. var for future use:

```
export FAASD_NODE_IP=<faasd-node-ip>
export DOCKER_USER=felipecruz
```

![faasd-node](docs/images/faasd-node.PNG)

Give `faasd` a few minutes to startup as the server will be initializing with the cloud-init configuration right after it has booted. If needed, you could SSH into the server and check the status of the `faasd` service. It should say `active (running).

```cli
ssh root@$FAASD_NODE_IP 'systemctl status faasd'

systemctl status faasd
● faasd.service - faasd
     Loaded: loaded (/lib/systemd/system/faasd.service; enabled; vendor preset: enabled)
     Active: active (running) since Mon 2020-12-28 15:14:15 CET; 12min ago
...
(ommited lines)
...
```

Then, access the OpenFaaS UI at http://$FAASD_NODE_IP:8080/ using the username `admin` and the random password generated previously.

![openfaas-ui](docs/images/openfaas-ui.PNG)

As you can see in the picture above, there are no functions deployed yet. Let's deploy our `spinner` function.

## Getting started

```cli
# Let's connect to our gateway
faas-cli login -g http://$FAASD_NODE_IP:8080 -p <password>
```

To only allow authenticated requests to our serverless function, we are going to use the Hetzner Cloud API token. Replace `HCLOUD_API_TOKEN` with yours.

```cli
echo -n HCLOUD_API_TOKEN > secret-api-key.txt
```

Create the secret:

```
faas-cli secret create secret-api-key \
  --from-file=secret-api-key.txt \
  -g http://$FAASD_NODE_IP:8080

Creating secret: secret-api-key
Created: 200 OK

# Check the secret has been created successfully
faas-cli secret list -g http://$FAASD_NODE_IP:8080

NAME
secret-api-key
```

Notice that the secret name is referenced in the [spinner.yml](spinner.yml#L16) file.

## Build, push and deploy the `spinner` function to our gateway

Let's replace the following placeholders defined in [spinner.yml](spinner.yml) with the actual faasd node IP and your username from DockerHub.

```bash
sed -i s/\$FAASD_NODE_IP/$FAASD_NODE_IP/g spinner.yml
sed -i s/\${DOCKER_USER}/$DOCKER_USER/g spinner.yml
```

Finally, build the image, push it to DockerHub and deploy it to the faasd node:

```cli
faas-cli up -f spinner.yml
```

The serverless function will serve the request by creating a new server of type `cx21` with image `Ubuntu 20.04` located on `nbg1` (Nuremberg) on Hetzner Cloud. Replace the `X-Api-Key` value with your own.

```cli
curl -v \
 --header "X-Api-Key: Ldf9LT..." \
 http://$FAASD_NODE_IP:8080/function/spinner?server_type=cx21&image_name=ubuntu-20.04&location=nbg1

...
(ommitted lines)
...
< HTTP/1.1 201 Created
< Content-Length: 53
< Content-Type: text/plain; charset=utf-8
< Date: Mon, 28 Dec 2020 15:54:49 GMT
< X-Duration-Seconds: 2.188351
<
* Connection #0 to host 168.119.167.108 left intact
Server '3374dea2-03ba-45b4-8331-3499e87be20e' created
```

## Build, push and deploy the `spinner-controller` function to our gateway

The `spinner-controller` serverless function is in charge of deleting servers when they are not needed anymore. This criteria could be one of the followings: `cpu`, `disk.0.iops.read`, `disk.0.iops.write`, `network.0.pps.in`, or `network.0.pps.out`.

Similarly, let's replace the following placeholders defined in [spinner-controller.yml](spinner-controller.yml) too with the actual faasd node IP and your username from DockerHub.

```bash
sed -i s/\$FAASD_NODE_IP/$FAASD_NODE_IP/g spinner-controller.yml
sed -i s/\${DOCKER_USER}/$DOCKER_USER/g spinner-controller.yml
```

Finally, build the image, push it to DockerHub and deploy it to the faasd node:

```cli
faas-cli up -f spinner-controller.yml
```

The serverless function will serve the request by checking if there are any running servers that are not doing any work based on the provided criteria.

For instance, if we want to delete any servers whose CPU load is below under 50% in the last 5 minutes, we would call:

```cli
curl -v \
 --header "X-Api-Key: ..." \
 http://$FAASD_NODE_IP:8080/function/spinner-controller?metric_name=cpu&metric_threshold=50&last_minutes=5
```

## Clean up

```cli
terraform destroy -var-file=main.tfvars -auto-approve
```
