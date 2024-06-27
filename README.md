# Luma VPN

This repository contains the source code for the Luma VPN app.

Please refer to our website luma.net for additional information how the service works.

## Features

- Proxy Protocols: SOCKS5, HTTP(S), Wireguard
- Rule-based Routing: dynamic scripting, domain, IP addresses, process name and more
- Server selection
- Lightweight GUI
- Custom DNS

## Usage

Create a `config.yaml` configuration file, and put it in the same directory as the luma binary:

```yaml
loglevel: debug
socks-port: 8787
listeners:
  - name: local-socks
    type: socks
    port: 10808
tun:
  enable: true
  stack: gvisor # system or gvisor
  device: tun://utun
```

## Installation

Build and install the `luma` binary

```shell
make luma
sudo cp ./build/luma /usr/local/bin
```

## Quick Start

Run luma with the given config and bind it to the primary interface.

```shell
luma -config config.yaml -proxy socks5://host:port -interface en0
```

## Download latest release

[去下载](https://github.com/lumavpn/luma/releases)

## Credits

- [v2ray/v2ray-core](https://github.com/v2ray/v2ray-core)
- [google/gvisor](https://github.com/google/gvisor)
- [xjasonlyu/tun2socks](https://github.com/xjasonlyu/tun2socks)

