#!/bin/bash
export PATH=/var/jb/usr/bin:/var/jb/usr/sbin:/usr/bin:/usr/sbin:/bin:/sbin:$PATH

TRUSTSTORE="/private/var/protected/trustd/private/TrustStore.sqlite3"
PREFS="/var/preferences/SystemConfiguration/preferences.plist"
CERT_PATH="/var/mobile/Media/sunny-ca.crt"
DEFAULT_HOST="192.168.31.215"
DEFAULT_PORT="2024"

print_usage() {
    echo "SunnyNet iOS Tool - One-click proxy setup"
    echo ""
    echo "Usage:"
    echo "  sunny-tool setup [host] [port]     - Install cert + set proxy"
    echo "  sunny-tool install-cert [path]     - Install CA cert"
    echo "  sunny-tool set-proxy <host> <port> - Set WiFi proxy"
    echo "  sunny-tool remove-proxy            - Remove WiFi proxy"
    echo "  sunny-tool status                  - Show status"
}

find_wifi_uuid() {
    plutil -convert xml1 "$PREFS" 2>/dev/null
    local UUID=$(grep -B20 "AirPort" "$PREFS" | grep "<key>[A-F0-9]" | tail -1 | sed 's/.*<key>//;s/<\/key>.*//')
    plutil -convert binary1 "$PREFS" 2>/dev/null
    echo "$UUID"
}

install_cert() {
    local cert_file="${1:-$CERT_PATH}"
    
    echo "[*] Installing certificate from $cert_file"
    
    if [ ! -f "$cert_file" ]; then
        echo "[!] Certificate file not found: $cert_file"
        exit 1
    fi
    
    # Convert PEM to DER
    openssl x509 -inform PEM -in "$cert_file" -outform DER -out /tmp/sunny-ca.der 2>/dev/null
    if [ $? -ne 0 ]; then
        # Maybe already DER
        cp "$cert_file" /tmp/sunny-ca.der
    fi
    
    # Get SHA-256
    local SHA256=$(openssl x509 -inform DER -in /tmp/sunny-ca.der -fingerprint -sha256 -noout 2>/dev/null | sed 's/.*=//;s/://g')
    echo "[*] SHA-256: $SHA256"
    
    # Get subject CN
    local SUBJ=$(openssl x509 -inform DER -in /tmp/sunny-ca.der -subject -noout 2>/dev/null)
    echo "[*] Subject: $SUBJ"
    
    # Create trust settings plist (always trust)
    cat > /tmp/sunny-tset.plist << 'TSETEOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<array>
    <dict>
        <key>kSecTrustSettingsAllowedError</key>
        <integer>-2147409654</integer>
        <key>kSecTrustSettingsPolicy</key>
        <data>KoZIhvdjZAYN</data>
        <key>kSecTrustSettingsResult</key>
        <integer>1</integer>
    </dict>
    <dict>
        <key>kSecTrustSettingsAllowedError</key>
        <integer>-2147409654</integer>
        <key>kSecTrustSettingsPolicy</key>
        <data>KoZIhvdjZAYL</data>
        <key>kSecTrustSettingsResult</key>
        <integer>1</integer>
    </dict>
</array>
</plist>
TSETEOF
    plutil -convert binary1 /tmp/sunny-tset.plist 2>/dev/null
    
    # Insert into TrustStore
    sqlite3 "$TRUSTSTORE" "INSERT OR REPLACE INTO tsettings (sha256, subj, tset, data) VALUES (X'$SHA256', X'00', readfile('/tmp/sunny-tset.plist'), readfile('/tmp/sunny-ca.der'));" 2>&1
    
    if [ $? -eq 0 ]; then
        killall trustd 2>/dev/null
        echo "[+] Certificate installed successfully!"
        echo "[+] trustd restarted"
    else
        echo "[!] Failed to install certificate"
        exit 1
    fi
}

set_proxy() {
    local host="${1:-$DEFAULT_HOST}"
    local port="${2:-$DEFAULT_PORT}"
    
    echo "[*] Setting WiFi proxy to $host:$port"
    
    local WIFI_UUID=$(find_wifi_uuid)
    if [ -z "$WIFI_UUID" ]; then
        echo "[!] Could not find WiFi service"
        exit 1
    fi
    echo "[*] WiFi UUID: $WIFI_UUID"
    
    plutil -convert xml1 "$PREFS" 2>/dev/null
    
    # Find the Proxies section for this WiFi service
    local START=$(grep -n "$WIFI_UUID" "$PREFS" | head -1 | cut -d: -f1)
    local PROXY_LINE=$(tail -n +$START "$PREFS" | grep -n "<key>Proxies</key>" | head -1 | cut -d: -f1)
    PROXY_LINE=$((START + PROXY_LINE - 1))
    
    local DICT_START=$((PROXY_LINE + 1))
    local END_LINE=$(tail -n +$DICT_START "$PREFS" | grep -n "</dict>" | head -1 | cut -d: -f1)
    END_LINE=$((DICT_START + END_LINE - 1))
    
    # Create new proxy block
    cat > /tmp/proxy_block.txt << PROXYEOF
			<key>Proxies</key>
			<dict>
				<key>ExceptionsList</key>
				<array>
					<string>*.local</string>
					<string>169.254/16</string>
				</array>
				<key>FTPPassive</key>
				<integer>1</integer>
				<key>HTTPEnable</key>
				<integer>1</integer>
				<key>HTTPPort</key>
				<integer>$port</integer>
				<key>HTTPProxy</key>
				<string>$host</string>
				<key>HTTPSEnable</key>
				<integer>1</integer>
				<key>HTTPSPort</key>
				<integer>$port</integer>
				<key>HTTPSProxy</key>
				<string>$host</string>
			</dict>
PROXYEOF
    
    # Replace old proxy block with new one
    sed -i "${PROXY_LINE},${END_LINE}d" "$PREFS"
    sed -i "$((PROXY_LINE-1))r /tmp/proxy_block.txt" "$PREFS"
    
    plutil -convert binary1 "$PREFS" 2>/dev/null
    killall -HUP configd 2>/dev/null
    
    echo "[+] WiFi proxy set to $host:$port"
}

remove_proxy() {
    echo "[*] Removing WiFi proxy..."
    
    local WIFI_UUID=$(find_wifi_uuid)
    if [ -z "$WIFI_UUID" ]; then
        echo "[!] Could not find WiFi service"
        exit 1
    fi
    
    plutil -convert xml1 "$PREFS" 2>/dev/null
    
    local START=$(grep -n "$WIFI_UUID" "$PREFS" | head -1 | cut -d: -f1)
    local PROXY_LINE=$(tail -n +$START "$PREFS" | grep -n "<key>Proxies</key>" | head -1 | cut -d: -f1)
    PROXY_LINE=$((START + PROXY_LINE - 1))
    
    local DICT_START=$((PROXY_LINE + 1))
    local END_LINE=$(tail -n +$DICT_START "$PREFS" | grep -n "</dict>" | head -1 | cut -d: -f1)
    END_LINE=$((DICT_START + END_LINE - 1))
    
    cat > /tmp/proxy_block.txt << 'PROXYEOF'
			<key>Proxies</key>
			<dict>
				<key>ExceptionsList</key>
				<array>
					<string>*.local</string>
					<string>169.254/16</string>
				</array>
				<key>FTPPassive</key>
				<integer>1</integer>
			</dict>
PROXYEOF
    
    sed -i "${PROXY_LINE},${END_LINE}d" "$PREFS"
    sed -i "$((PROXY_LINE-1))r /tmp/proxy_block.txt" "$PREFS"
    
    plutil -convert binary1 "$PREFS" 2>/dev/null
    killall -HUP configd 2>/dev/null
    
    echo "[+] Proxy removed!"
}

show_status() {
    echo "[*] SunnyNet iOS Status"
    echo "-----------------------"
    
    # Check cert
    local CERT_CHECK=$(sqlite3 "$TRUSTSTORE" "SELECT hex(sha256) FROM tsettings;" 2>/dev/null)
    if echo "$CERT_CHECK" | grep -qi "1A61D4F7"; then
        echo "[+] CA Certificate: Installed"
    else
        echo "[-] CA Certificate: Not installed"
    fi
    
    # Check proxy
    plutil -convert xml1 "$PREFS" 2>/dev/null
    if grep -q "HTTPEnable" "$PREFS"; then
        local PROXY_HOST=$(grep -A1 "<key>HTTPProxy</key>" "$PREFS" | grep string | sed 's/.*<string>//;s/<\/string>.*//')
        local PROXY_PORT=$(grep -A1 "<key>HTTPPort</key>" "$PREFS" | grep integer | sed 's/.*<integer>//;s/<\/integer>.*//')
        echo "[+] WiFi Proxy: $PROXY_HOST:$PROXY_PORT"
    else
        echo "[-] WiFi Proxy: Not configured"
    fi
    plutil -convert binary1 "$PREFS" 2>/dev/null
}

# Main
case "${1:-}" in
    setup)
        echo "[*] SunnyNet One-Click Setup"
        echo "[*] ========================"
        install_cert "${4:-$CERT_PATH}"
        set_proxy "${2:-$DEFAULT_HOST}" "${3:-$DEFAULT_PORT}"
        echo ""
        echo "[+] Setup complete! All traffic now goes through SunnyNet"
        ;;
    install-cert)
        install_cert "$2"
        ;;
    set-proxy)
        set_proxy "$2" "$3"
        ;;
    remove-proxy)
        remove_proxy
        ;;
    status)
        show_status
        ;;
    *)
        print_usage
        ;;
esac
