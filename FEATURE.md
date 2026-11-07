# Sign in with NetBird

If your device is connected to your NetBird network, you can sign in to your apps through Pocket ID with a single click. No passkey prompt, Pocket ID already knows who you are because NetBird knows which user your device belongs to. Everyone else keeps signing in with their passkeys like before.

It's similar to what Tailscale's tsidp does, but it works with the Pocket ID you already have.

This feature needs changes in three projects:

- **NetBird server and reverse proxy**: [`feat/tunnel-identity-passthrough`](https://github.com/netbirdio/netbird/compare/main...stonith404:netbird:feat/tunnel-identity-passthrough). The proxy tells the app which user owns the device, and devices in a public service's access groups reach it through the tunnel.
- **NetBird dashboard**: [`feat/public-service-access-groups`](https://github.com/netbirdio/dashboard/compare/main...stonith404:netbird-dashboard:feat/public-service-access-groups). Lets you set access groups on public services.
- **Pocket ID**: [`poc/netbird-auth`](https://github.com/pocket-id/pocket-id/compare/main...poc/netbird-auth). Adds the NetBird sign-in to the authorize page.

## How it works

Pocket ID runs behind NetBird's reverse proxy, on the same domain as always.

- Devices in the groups you choose resolve the Pocket ID domain to the proxy's NetBird IP, so their requests go through the tunnel. NetBird then tells Pocket ID which user owns the device.
- Everyone else resolves the public IP and doesn't notice anything.

You'll then see a **Continue as … via NetBird** button when an app sends you to Pocket ID to sign in.

A few things to keep in mind:

- A NetBird sign-in can only be used to sign in to apps. To manage your account, passkeys or settings, you still need your passkey.
- Apps that require reauthentication still ask for a passkey.

## Requirements

- NetBird uses Pocket ID as its identity provider, either directly or as a connector of NetBird's embedded IdP. That's how Pocket ID knows which NetBird user is which Pocket ID user.
- The NetBird server and reverse proxy from [stonith404/netbird](https://github.com/stonith404/netbird) (`feat/tunnel-identity-passthrough`) and the dashboard from [stonith404/netbird-dashboard](https://github.com/stonith404/netbird-dashboard) (`feat/public-service-access-groups`):
  - `ghcr.io/stonith404/netbird-server:tunnel-identity`
  - `ghcr.io/stonith404/reverse-proxy:tunnel-identity`
  - `ghcr.io/stonith404/dashboard:tunnel-identity`
- A Pocket ID version with NetBird support.
- Pocket ID is published as a service on the NetBird reverse proxy, and its domain is the cluster domain or a custom domain you added in NetBird.

## Setup

### 1. Enable private mode on the reverse proxy

Set this on the reverse proxy that serves Pocket ID:

```yaml
environment:
  NB_PROXY_PRIVATE: "true"
```

This makes the proxy listen on its NetBird IP as well, so devices can reach it through the tunnel.

### 2. Generate a secret

Pocket ID needs a way to know that a request really went through the NetBird proxy. For that, the proxy sends a secret header that only it and Pocket ID know.

```sh
openssl rand -hex 32
```

### 3. Configure the service in NetBird

Open the Pocket ID service in the NetBird dashboard and make sure:

- No authentication is enabled on the service.
- The target has a custom header `X-NetBird-Proxy-Secret` with the secret from step 2.
- Under **Authentication → NetBird Tunnel Access**, select the groups whose devices should get the button.

### 4. Configure Pocket ID

```yaml
environment:
  NETBIRD_PROXY_SECRET: <secret from step 2>
```

`NETBIRD_PROXY_SECRET_FILE` works as well.

### 5. Try it

From a device in one of the access groups:

```sh
curl https://id.example.com/api/netbird/identity
```

It should return your user. Then sign in to one of your apps and you should see the NetBird button.
