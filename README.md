# Mayfly
A proof of concept for a simple, ephemeral personal-use VPN service.

Mayfly exists in response to growing government pressure on VPN providers, including the UK's potential moves toward restricting commercial VPN access. The argument behind those restrictions assumes that VPN access is something that can be switched off at the provider level. Mayfly demonstrates that it cannot: anyone with a VPS and basic technical knowledge can provision their own VPN in seconds and tear it down just as quickly.

## How it works
Mayfly SSHes into a VPS you already control, spins up a minimal WireGuard server inside a Docker container, and generates a WireGuard client config you can connect with immediately. When you're done, the container is removed and every cryptographic trace of the session is gone. No persistent keys, no permanent server config, no logs of what IP you were assigned.

- **Ephemeral by design** - fresh WireGuard keypairs are generated for every session and never written to the server
- **Self-hosted** - you own the VPS, you own the connection, no third party is involved
- **Minimal footprint** - the server side is a single Alpine container; the host OS is left untouched