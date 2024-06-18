![LumaVPN](docs/icon.png)
# LumaVPN

[![GitHub Workflow][1]](https://github.com/lumavpn/luma/actions)
[![Go Version][2]](https://github.com/lumavpn/luma/blob/main/go.mod)

[1]: https://img.shields.io/github/actions/workflow/status/lumavpn/luma/dev.yml?logo=github
[2]: https://img.shields.io/github/go-mod/go-version/lumavpn/luma?logo=go

This repository contains the source code for the Luma VPN app.

Please refer to our website luma.net for additional information how the service works.

## Features

- Proxy protocols: SOCKS5, HTTP(S), Wireguard
- Full Platform support: Linux, MacOS, Windows, Android
- Server selection
- Lightweight GUI
- Full IPv6 support
- Network stack
- Split tunneling
- Custom DNS

## Usage

Create a `config.yaml` configuration file, and put it in same directory as the luma binary:

```yaml
loglevel: debug
listeners:
  - name: local-socks
    type: socks
    port: 10808
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
- [Dreamacro/clash](https://github.com/Dreamacro/clash)
- [sagernet/sing-tun](https://github.com/sagernet/sing-tun)
