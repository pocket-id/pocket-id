If you are using [Pocket ID](https://pocket-id.org/) with a **Cloudflare** associated domain, and would like to install Nginx on the same host, with auto-renewing Let's Encrypt Certificates, here you go.


Install [Pocket ID Proxmox VE Helper-Script](https://community-scripts.github.io/ProxmoxVE/scripts?id=pocketid) like normal and then edit the internal caddy file: 
```
sudo nano /etc/caddy/Caddyfile
```
Change first line from `:{$CADDY_PORT:80} {` to `:{$CADDY_PORT:8180} {`  (or any other port)

Reboot

Install modified CloudflareNginx Script 
```
curl -sLO https://raw.githubusercontent.com/taslabs-net/CloudflareNginx/main/installwithpocketid.sh
```

Make it executable: 
```
chmod +x installwithpocketid.sh
```

Run script
```
sudo ./installwithpocketid.sh
```

What happens:  

SSL Configuration:
Creates secure Cloudflare credential file
Generates Let's Encrypt certificate using DNS challenge
Auto renews cert every 30 days

Nginx Setup:
Creates optimized SSL configuration
Sets up HTTPS redirect
Configures reverse proxy with WebSocket support and increased buffer limits
