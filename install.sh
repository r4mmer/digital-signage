#!/usr/bin/env bash

set -e

echo "Digital Signage Installation Script"
echo "==================================="

install_deps() {
    sudo apt-get update
    sudo apt-get install -y chromium-browser unclutter wget curl
}

# Detect Raspberry Pi model
detect_pi_model() {
    local model;
    model=$(grep "Model" /proc/cpuinfo | head -1 | cut -d: -f2 | xargs)

    if [[ "$model" == *"Pi Zero"* ]]; then
        echo "pi-zero"
    elif [[ "$model" == *"Pi 3"* ]]; then
        echo "pi-3"
    elif [[ "$model" == *"Pi 4"* ]] || [[ "$model" == *"Pi 400"* ]]; then
        echo "pi-4"
    else
        echo "unknown"
    fi
}

PI_MODEL=$(detect_pi_model)
echo "Detected Raspberry Pi model: $PI_MODEL"

if [ "$PI_MODEL" == "unknown" ]; then
    echo "Warning: Unknown Raspberry Pi model. Defaulting to Pi 4 binary."
    PI_MODEL="pi-4"
fi

install_deps

# Create application directory
APP_DIR="/home/pi/digital-signage"
echo "Creating application directory: $APP_DIR"
mkdir -p "$APP_DIR"
mkdir -p "$APP_DIR/media"

echo "Installing binary and files..."
# Download tarball 
LOCAL_FILE=digital_signage.gz.tar

curl -s https://api.github.com/repos/r4mmer/digital-signage/releases/latest \
    | grep browser_download_url | grep "$PI_MODEL" \
    | cut -d : -f 2,3 | tr -d \" \
    | wget -O "$LOCAL_FILE" -qi - ;
# Open file contents
tar -xzf "$LOCAL_FILE" -C "$APP_DIR"
rm "$LOCAL_FILE"

# Set ownership
sudo chown -R pi:pi "$APP_DIR"
chmod +x "$APP_DIR/digital-signage"
chmod +x "$APP_DIR/kiosk.sh"

# Install systemd service
echo "Installing systemd service..."

cat << EOF > /etc/systemd/system/digital-signage.service
[Unit]
Description=Digital Signage Application
After=network.target
Wants=network.target

[Service]
Type=simple
User=pi
Group=pi
WorkingDirectory=/home/pi/digital-signage
ExecStart=/home/pi/digital-signage/digital-signage
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

# Environment variables
Environment=MEDIA_DIR=/home/pi/digital-signage/media
Environment=PORT=8080
Environment=S3_BUCKET=
Environment=S3_REGION=sa-east-1
Environment=SYNC_INTERVAL_MINUTES=15
Environment=AWS_ACCESS_KEY_ID=your-access-key
Environment=AWS_SECRET_ACCESS_KEY=your-secret-key

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/home/pi/digital-signage

[Install]
WantedBy=multi-user.target
EOF

## AUTOSTART KIOSK
mkdir -p /home/pi/.config/lxsession/LXDE-pi/
cat << EOF > /home/pi/.config/lxsession/LXDE-pi/autostart
@lxpanel --profile LXDE-pi
@pcmanfm --desktop --profile LXDE-pi
@lxterminal -e bash /home/pi/digital-signage/kiosk.sh
EOF

sudo systemctl daemon-reload
sudo systemctl enable digital-signage

# Disabling bluetooth, usb and eth ???
# echo 'dtoverlay=disable-bt' >> /boot/firmware/config.txt
# echo '1-1' | sudo tee /sys/bus/usb/drivers/usb/unbind
# disable bluetooth again?
sudo systemctl disable bluetooth

echo "Installation completed!"
echo ""
echo "Next steps:"
echo "1. Add your video files to: $APP_DIR/media/"
echo "2. If using S3 sync, edit $APP_DIR/.env with your S3 configuration"
echo "3. Start the service: sudo systemctl start digital-signage"
echo "4. Check status: sudo systemctl status digital-signage"
echo "5. View logs: sudo journalctl -u digital-signage -f"
echo ""
echo "The web interface will be available at http://localhost:8080"
