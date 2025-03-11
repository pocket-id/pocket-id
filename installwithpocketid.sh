#!/bin/bash
# CloudNginx Installer with WebSocket Support and Persistence

# Configuration
BLUE='\033[0;34m'
NC='\033[0m'
DEFAULT_PORT="443"
CLOUDFLARE_CRED_PATH="/etc/letsencrypt/cloudflare.ini"
TMP_DIR=$(mktemp -d)

cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

show_header() {
    clear
    echo -e "${BLUE}"
    echo "==============================================="
    echo "        Cloudflare Nginx Automated Setup"
    echo "==============================================="
    echo -e "${NC}"
}

check_root() {
    [ "$EUID" -eq 0 ] || { echo "Please run as root"; exit 1; }
}

ask_question() {
    echo -e "${BLUE}"
    read -r -p "$1: " ${2}
    echo -e "${NC}"
}

validate_domain() {
    [[ "$1" =~ ^([a-zA-Z0-9](-?[a-zA-Z0-9])*\.)+[a-zA-Z]{2,}$ ]] || {
        echo "Invalid domain format"; exit 1
    }
}

install_core_dependencies() {
    echo -e "${BLUE}Updating system packages...${NC}"
    apt-get update -qq && apt-get upgrade -y -qq
    
    echo -e "${BLUE}Installing required components...${NC}"
    apt-get install -y -qq \
        nginx \
        python3-certbot-dns-cloudflare \
        curl \
        ufw
}

handle_cloudflare_credentials() {
    ask_question "Enter your Cloudflare Email" CF_EMAIL
    ask_question "Enter your Cloudflare API Key" CF_API_KEY
    ask_question "Enter your Hostname (FQDN)" DOMAIN
    
    validate_domain "$DOMAIN"
    
    mkdir -p $(dirname "$CLOUDFLARE_CRED_PATH")
    cat > "$CLOUDFLARE_CRED_PATH" <<EOF
dns_cloudflare_email = ${CF_EMAIL}
dns_cloudflare_api_key = ${CF_API_KEY}
EOF
    chmod 600 "$CLOUDFLARE_CRED_PATH"
}

generate_ssl() {
    echo -e "${BLUE}Generating SSL certificate...${NC}"
    certbot certonly --dns-cloudflare \
        --dns-cloudflare-credentials "$CLOUDFLARE_CRED_PATH" \
        -d "$DOMAIN" \
        --non-interactive \
        --agree-tos \
        --email "$CF_EMAIL"
}

configure_nginx() {
    local DOMAIN=$1
    local PORT=$2
    echo -e "${BLUE}Configuring NGINX for ${DOMAIN}...${NC}"
    # Create full nginx configuration
    cat > "/etc/nginx/sites-available/${DOMAIN}" <<EOF
server {
    listen 80;
    server_name ${DOMAIN};
    # Redirect HTTP to HTTPS
    return 301 https://\$host\$request_uri;
}
server {
    listen 443 ssl;
    server_name ${DOMAIN};
    # SSL Configuration
    ssl_certificate /etc/letsencrypt/live/${DOMAIN}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${DOMAIN}/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers on;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384;
    # OCSP Stapling
    ssl_stapling on;
    ssl_stapling_verify on;
    # Proxy configuration
    location / {
        proxy_pass http://127.0.0.1:${PORT};
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_cache_bypass \$http_upgrade;

        # Additional proxy buffer settings
        proxy_busy_buffers_size   512k;
        proxy_buffers            4 512k;
        proxy_buffer_size         256k;
    }
}
EOF
    # Enable site configuration
    ln -sf "/etc/nginx/sites-available/${DOMAIN}" "/etc/nginx/sites-enabled/"
    nginx -t && systemctl reload nginx
}

configure_firewall() {
    if command -v ufw >/dev/null 2>&1; then
        ufw allow 80/tcp
        ufw allow 443/tcp
        ufw reload
    fi
}

ensure_service_persistence() {
    echo -e "${BLUE}Ensuring service persistence...${NC}"
    
    # Enable and start Nginx if not already active
    if ! systemctl is-active --quiet nginx; then
        systemctl enable --now nginx
    fi
    
    # Enable Certbot renewal timer
    if systemctl list-timers | grep -q certbot; then
        systemctl enable --now certbot.timer
    fi
}

setup_certbot_renewal() {
    echo -e "${BLUE}Setting up Certbot renewal...${NC}"
    
    # Create renewal hook
    mkdir -p /etc/letsencrypt/renewal-hooks/deploy
    cat > /etc/letsencrypt/renewal-hooks/deploy/nginx-reload.sh <<EOF
#!/bin/bash
systemctl reload nginx
EOF
    
    chmod +x /etc/letsencrypt/renewal-hooks/deploy/nginx-reload.sh
    
    # Test renewal
    if certbot renew --dry-run; then
        echo -e "${BLUE}Certificate renewal configured successfully${NC}"
    else
        echo -e "${BLUE}Certificate renewal test failed${NC}"
        exit 1
    fi
}

main() {
    show_header
    check_root
    handle_cloudflare_credentials
    install_core_dependencies
    generate_ssl
    setup_certbot_renewal
    
    ask_question "Enter your application port (default: 8180)" PORT
    PORT=${PORT:-8180}
    
    configure_nginx "$DOMAIN" "$PORT"
    configure_firewall
    ensure_service_persistence
    
    echo -e "${BLUE}Setup completed successfully!${NC}"
    echo -e "${BLUE}Access your site at: https://${DOMAIN}${NC}"
    
    # Display important information
    echo -e "\n${BLUE}Important Notes:${NC}"
    echo -e "1. Your Cloudflare credentials are stored securely at ${CLOUDFLARE_CRED_PATH}"
    echo -e "2. SSL certificates will auto-renew before expiration"
    echo -e "3. Nginx is configured to start automatically on boot"
    echo -e "4. Firewall rules (if UFW is present) are persistent"
}

main
