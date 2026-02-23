# Digital Signage for Raspberry Pi

A simple, lightweight digital signage application built in Go that plays video content in a loop and syncs with S3 storage.

## Features

- **Video Playback**: Automatically plays video files from a directory in sequence
- **S3 Sync**: Background synchronization with Amazon S3 bucket for remote content updates
- **Web Interface**: Clean, fullscreen web interface optimized for kiosk mode
- **Lightweight**: Uses Go standard library, minimal dependencies
- **Auto-restart**: Robust error handling and automatic recovery

## Quick Start

### Development Setup (NixOS)

```bash
# Enter development environment
nix develop

# Run locally for testing
go run .
```

### Raspberry Pi Deployment

1. **Build the application** (on your development machine):
   ```bash
   ./build.sh
   ```

2. **Transfer files to Raspberry Pi**:
   ```bash
   scp -r build/ digital-signage.service install.sh pi@your-pi-ip:~/
   ```

3. **Install on Raspberry Pi**:
   ```bash
   ssh pi@your-pi-ip
   chmod +x install.sh
   ./install.sh
   ```

4. **Add your media files**:
   ```bash
   # Copy video files to the media directory
   cp your-videos/* /home/pi/digital-signage/media/
   ```

5. **Start the service**:
   ```bash
   sudo systemctl start digital-signage
   ```

## Raspberry Pi Kiosk Mode Setup

### 1. Update System
```bash
sudo apt update && sudo apt upgrade -y
```

### 2. Install Required Packages
```bash
sudo apt install -y chromium-browser unclutter xdotool
```

### 3. Configure Auto-login
```bash
sudo raspi-config
# Navigate to: System Options -> Boot / Auto Login -> Desktop Autologin
```

### 4. Create Kiosk Startup Script
```bash
mkdir -p /home/pi/.config/autostart
```

Create `/home/pi/.config/autostart/kiosk.desktop`:
```ini
[Desktop Entry]
Type=Application
Name=Kiosk
Exec=/home/pi/kiosk.sh
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
```

Create `/home/pi/kiosk.sh`:
```bash
#!/bin/bash

# Wait for network and digital signage service
sleep 10

# Hide cursor
unclutter -idle 0 &

# Start Chromium in kiosk mode
chromium-browser \
  --kiosk \
  --no-sandbox \
  --disable-infobars \
  --disable-features=TranslateUI \
  --disable-component-extensions-with-background-pages \
  --disable-background-networking \
  --disable-sync \
  --disable-translate \
  --hide-scrollbars \
  --disable-web-security \
  --disable-features=VizDisplayCompositor \
  --autoplay-policy=no-user-gesture-required \
  http://localhost:8080
```

Make it executable:
```bash
chmod +x /home/pi/kiosk.sh
```

### 5. Disable Screen Blanking
Add to `/boot/config.txt`:
```
# Disable screen blanking
hdmi_blanking=1
```

Add to `/etc/xdg/lxsession/LXDE-pi/autostart`:
```
@xset s off
@xset -dpms
@xset s noblank
```

### 6. Configure Boot Options
```bash
sudo raspi-config
# Navigate to: Advanced Options -> Memory Split -> 128
# Navigate to: Advanced Options -> GL Driver -> G1 GL (Fake KMS)
```

## Configuration

### Environment Variables

The application can be configured using environment variables:

- `MEDIA_DIR`: Directory containing video files (default: `./media`)
- `PORT`: HTTP server port (default: `8080`)
- `S3_BUCKET`: S3 bucket name for sync (optional)
- `S3_REGION`: AWS region (default: `us-east-1`)
- `SYNC_INTERVAL_MINUTES`: How often to sync with S3 (default: `15`)
- `AWS_ACCESS_KEY_ID`: AWS access key (optional, can use credential files)
- `AWS_SECRET_ACCESS_KEY`: AWS secret key (optional, can use credential files)

### S3 Sync Setup

1. **Create S3 Bucket**:
   ```bash
   aws s3 mb s3://your-signage-bucket
   ```

2. **Configure AWS Credentials**:
   ```bash
   # Option 1: Using AWS CLI
   aws configure
   
   # Option 2: Environment variables in .env file
   echo "AWS_ACCESS_KEY_ID=your-key" >> /home/pi/digital-signage/.env
   echo "AWS_SECRET_ACCESS_KEY=your-secret" >> /home/pi/digital-signage/.env
   ```

3. **Enable S3 Sync**:
   Edit `/home/pi/digital-signage/.env`:
   ```bash
   S3_BUCKET=your-signage-bucket
   S3_REGION=us-east-1
   SYNC_INTERVAL_MINUTES=15
   ```

4. **Upload Media to S3**:
   ```bash
   aws s3 cp your-video.mp4 s3://your-signage-bucket/
   ```

## Supported Video Formats

- MP4 (.mp4)
- AVI (.avi)
- MOV (.mov)
- MKV (.mkv)
- WebM (.webm)
- M4V (.m4v)
- 3GP (.3gp)

## Troubleshooting

### Service Won't Start

1. Check logs:
   ```bash
   sudo journalctl -u digital-signage -n 50
   ```

2. Verify binary permissions:
   ```bash
   ls -la /home/pi/digital-signage/digital-signage
   ```

3. Test manually:
   ```bash
   cd /home/pi/digital-signage
   ./digital-signage
   ```

### No Videos Playing

1. Check media directory:
   ```bash
   ls -la /home/pi/digital-signage/media/
   ```

2. Verify file formats (must be supported video formats)

3. Check application logs for errors

### S3 Sync Issues

1. Verify AWS credentials:
   ```bash
   aws s3 ls s3://your-bucket-name
   ```

2. Check S3 configuration in `.env` file

3. Monitor sync logs:
   ```bash
   sudo journalctl -u digital-signage -f | grep -i s3
   ```

### Kiosk Mode Issues

1. Check if Chromium is running:
   ```bash
   ps aux | grep chromium
   ```

2. Test the web interface manually:
   ```bash
   chromium-browser http://localhost:8080
   ```

3. Verify autostart configuration:
   ```bash
   ls -la /home/pi/.config/autostart/
   cat /home/pi/.config/autostart/kiosk.desktop
   ```

### Performance Optimization

For better performance on older Raspberry Pi models:

1. **Reduce video resolution** - Use 720p or lower for Pi Zero/3
2. **Optimize video encoding** - Use H.264 baseline profile
3. **Increase GPU memory split**:
   ```bash
   sudo raspi-config
   # Advanced Options -> Memory Split -> 128 or 256
   ```
4. **Disable unnecessary services**:
   ```bash
   sudo systemctl disable bluetooth
   sudo systemctl disable wifi-powersave-off
   ```

## Development

### Building Locally

```bash
# Enter Nix development shell
nix develop

# Install dependencies
go mod tidy

# Run locally
go run .

# Build for current platform
go build .
```
