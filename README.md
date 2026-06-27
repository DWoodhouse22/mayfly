# Mayfly VPN
A proof of concept for a simple, ephemeral personal-use VPN service.

Mayfly exists in response to growing government pressure on VPN providers, including the UK's potential moves toward restricting commercial VPN access. The argument behind those restrictions assumes that VPN access is something that can be switched off at the provider level. Mayfly demonstrates that it cannot: anyone with a VPS and basic technical knowledge can provision their own VPN in seconds and tear it down just as quickly.

## Supported Platforms
| OS      | Architecture    | Status      |
|---------|-----------------|-------------|
| Windows | amd64           | ✓           |
| Windows | arm64, arm, x86 | Untested    |
| Linux   | amd64           | Coming soon |
| Linux   | arm64           | Coming soon |

>Note: MacOS is unlikely to be supported as this is a free, proof-of-concept project. MacOS users can try [v0.1.1-alpha](https://github.com/DWoodhouse22/mayfly/releases/tag/v0.1.1-alpha) which requires manually installing WireGuard

## How it works
Mayfly SSHes into a VPS you already control, spins up a minimal WireGuard server inside a Docker container, and automatically establishes a WireGuard tunnel. When you're done, the container is removed and every cryptographic trace of the session is gone. No persistent keys, no permanent server config, no logs of what IP you were assigned.

- **Ephemeral by design** - fresh WireGuard keypairs are generated for every session and never written to the server
- **Self-hosted** - you own the VPS, you own the connection, no third party is involved
- **Minimal footprint** - the server side is a single Alpine container; the host OS is left untouched

## VPS Setup
These steps are required once on any VPS you intend to run Mayfly.

**1. Install Docker**  
[Docker installation instructions](https://docs.docker.com/engine/install/)

**2. Configure the firewall (recommended)**
A firewall is not required for Mayfly to function - on a fresh VPS with no active firewall, port 51820 is reachable by default. However, enabling one is strongly recommended to limit your VPS's attack surface to only the ports it needs.

```
ufw allow ssh
ufw allow 51820/udp
ufw enable
```

Two pitfalls to be aware of:
- If UFW is already active on your VPS but port 51820/udp has not been allowed, WireGuard clients will silently fail to connect - the `mayfly server start` command will succeed but the tunnel won't pass traffic.
- If you enable UFW without first running `ufw allow ssh`, you will lock yourself out of the VPS.

## Usage
>Note: Windows requires running as Administrator, and Linux requires `sudo`  

**Start a session**
```
mayfly server start --host <vps-ip> --user <ssh-user> --token <random-string>
```
Specify an SSH key with `--key path/to/key-file`. It will default to standard SSH key names: `id_ed25519`, `id_ecdsa`, or `id_rsa`.  
`--token <random-string>` required. This is the token you will use to add secondary clients to the same VPN.  
This SSHes into your VPS, pulls and starts the container, and creates the WireGuard tunnel. The process blocks until Ctrl+C which disconnects the tunnel and removes the container.

**Stop a session**  
Ctrl+C on the blocking `mayfly server start` process.  
Fallback for unclean shutdowns is:

```
mayfly server stop --host <vps-ip> --user <ssh-user>
```