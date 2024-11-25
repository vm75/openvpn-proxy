## IMPORTANT NOTE: THIS REPO IS DEPRECATED. PLEASE TRY [https://github.com/vm75/vpn-sandbox](https://github.com/vm75/vpn-sandbox) WHICH IS WRITEN IN GO FROM SCRATCH.

# OpenVPN Client for Docker
## What is this and what does it do?
[`vm75/openvpn-proxy`](https://hub.docker.com/r/vm75/openvpn-proxy)  is a containerized OpenVPN client. It has a kill switch built with `iptables` that kills Internet connectivity to the container if the VPN tunnel goes down for any reason. It also includes an HTTP proxy server ([Tinyproxy](https://tinyproxy.github.io/)) and a SOCKS proxy server ([Dante](https://www.inet.no/dante/index.html)). This allows hosts and non-containerized applications to use the VPN without having to run VPN clients on those hosts.

This image requires you to supply the necessary OpenVPN configuration file(s). Because of this, any VPN provider should work (however, if you find something that doesn't, please open an issue for it).
The image monitors for changes to the configuration files and updates the vpn connection automatically. If VPN fails to connect, the kill switch is activated.

## Why?
Having a containerized VPN client lets you use container networking to easily choose which applications you want using the VPN instead of having to set up split tunnelling. It also keeps you from having to install an OpenVPN client on the underlying host.

The idea for this image came from a similar project by [qdm12](https://github.com/qdm12) that has since evolved into something bigger and more complex than I wanted to use. I decided to dissect it and take it in my own direction. I plan to keep everything here well-documented because I want this to be a learning experience for both me and hopefully anyone else that uses it.

## How do I use it?
### Getting the image
You can either pull it from GitHub Container Registry or build it yourself.

To pull from GitHub Container Registry, run `docker pull vm75/openvpn-proxy`.

To build it yourself, do the following:
```bash
git clone https://github.com/vm75/openvpn-proxy.git
cd openvpn-proxy
docker build -t vm75/openvpn-proxy .
```

### Creating and running a container
The image requires the container be created with the `NET_ADMIN` and `NET_RAW` capabilities and `/dev/net/tun` accessible. Below are bare-bones examples for `docker run` and Compose; however, you'll probably want to do more than just run the VPN client. See the sections below to learn how to use the [proxies](#http_proxy-and-socks_proxy) and have [other containers use `openvpn-proxy`'s network stack](#using-with-other-containers).

Create files vpn.ovpn & vpn.auth in a data folder. Store your VPN provider's ovpn file in vpn.ovpn & your credentials in vpn.auth (username=1st line, password=2nd line)

#### `docker run`
```bash
docker run -d \
  --name=openvpn-proxy \
  --cap-add=NET_ADMIN --cap-add=NET_RAW \
  --device=/dev/net/tun \
  -v <path to folder with vpn files>:/data \
  vm75/openvpn-proxy
```

#### `docker-compose`
```yaml
version: '2'

  vpn:
    container_name: vpn
    image: vm75/openvpn-proxy:latest
    cap_add:
      - NET_ADMIN
      - NET_RAW
    environment:
      - PUID=${PUID}
      - PGID=${PGID}
      - KILL_SWITCH=on
      - SOCKS_PROXY=on
      - HTTP_PROXY=on
      - SUBNETS=192.168.1.0/24,10.89.0.0/16
    dns:
      - 1.1.1.1
      - 1.0.0.1
    devices:
      - /dev/net/tun
    ports:
      - 4444:8080 # http-proxy
      - 5555:1080 # socks-proxy
    volumes:
      - <path to folder with vpn files>:/data
    restart: unless-stopped
```

#### Environment variables
| Variable | Default (blank is unset) | Description |
| --- | --- | --- |
| `SUBNETS` | | A list of one or more comma-separated subnets (e.g. `192.168.0.0/24,192.168.1.0/24`) to allow outside of the VPN tunnel. See important note about this [below](#subnets). |
| `VPN_LOG_LEVEL` | `3` | OpenVPN verbosity (`1`-`11`) |
| `HTTP_PROXY` | `off` | The on/off status of Tinyproxy, the built-in HTTP proxy server. To enable, set to `on`. Any other value (including unset) will cause the proxy server to not start. It listens on port 8080. |
| `SOCKS_PROXY` | `off` | The on/off status of Dante, the built-in SOCKS proxy server. To enable, set to `on`. Any other value (including unset) will cause the proxy server to not start. It listens on port 1080. |

##### Environment variable considerations
###### `SUBNETS`
**Important**: The DNS server used by this container prior to VPN connection must be included in the value specified. For example, if your container is using 192.168.1.1 as a DNS server, then this address or an appropriate CIDR block must be included in `SUBNETS`. This is necessary because the kill switch blocks traffic outside of the VPN tunnel before it's actually established. If the DNS server is not allowed, the server addresses in the VPN configuration will not resolve.

The subnets specified will be allowed through the firewall which allows for connectivity to and from hosts on the subnets.

###### `HTTP_PROXY` and `SOCKS_PROXY`
If enabling the the proxy server(s), you'll want to publish the appropriate port(s) in order to access the server(s). To do that using `docker run`, add `-p <host_port>:8080` and/or `-p <host_port>:1080` where `<host_port>` is whatever port you want to use on the host. If you're using `docker-compose`, add the relevant port specification(s) from the snippet below to the `openvpn-proxy` service definition in your Compose file.
```yaml
ports:
    - <host_port>:8080
    - <host_port>:1080
```

### Using with other containers
Once you have your `openvpn-proxy` container up and running, you can tell other containers to use `openvpn-proxy`'s network stack which gives them the ability to utilize the VPN tunnel. There are a few ways to accomplish this depending how how your container is created.

If your container is being created with
1. the same Compose YAML file as `openvpn-proxy`, add `network_mode: service:openvpn-proxy` to the container's service definition.
2. a different Compose YAML file than `openvpn-proxy`, add `network_mode: container:openvpn-proxy` to the container's service definition.
3. `docker run`, add `--network=container:openvpn-proxy` as an option to `docker run`.

Once running and provided your container has `wget` or `curl`, you can run `docker exec <container_name> wget -qO - ifconfig.me` or `docker exec <container_name> curl -s ifconfig.me` to get the public IP of the container and make sure everything is working as expected. This IP should match the one of `openvpn-proxy`.

#### Handling ports intended for connected containers
If you have a connected container and you need to access a port that container, you'll want to publish that port on the `openvpn-proxy` container instead of the connected container. To do that, add `-p <host_port>:<container_port>` if you're using `docker run`, or add the below snippet to the `openvpn-proxy` service definition in your Compose file if using `docker-compose`.
```yaml
ports:
    - <host_port>:<container_port>
```
In both cases, replace `<host_port>` and `<container_port>` with the port used by your connected container.

### Verifying functionality
Once you have container running `vm75/openvpn-proxy`, run the following command to spin up a temporary container using `openvpn-proxy` for networking. The `wget -qO - ifconfig.me` bit will return the public IP of the container (and anything else using `openvpn-proxy` for networking). You should see an IP address owned by your VPN provider.
```bash
docker run --rm -it --network=container:openvpn-proxy alpine wget -qO - ifconfig.me
```

