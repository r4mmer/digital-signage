#!/bin/bash

# Digital Signage Kiosk Mode Script
# This script configures Chromium to run in full kiosk mode

# Wait for desktop environment and network
sleep 15

# Kill any existing Chromium processes
pkill -f chromium-browser || true

# Wait a moment for processes to terminate
sleep 2

# Hide mouse cursor when idle
unclutter -idle 0.1 -root &

# Disable screen blanking and power management
xset s off
xset -dpms
xset s noblank

# Start Chromium in kiosk mode
chromium-browser \
  --kiosk \
  --start-fullscreen \
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
  --no-first-run \
  --fast \
  --fast-start \
  --disable-default-apps \
  --disable-popup-blocking \
  --disable-prompt-on-repost \
  --no-message-box \
  --disable-hang-monitor \
  --disable-logging \
  --disable-client-side-phishing-detection \
  --disable-component-update \
  --disable-default-apps \
  --disable-dev-shm-usage \
  --no-zygote \
  --memory-pressure-off \
  --max_old_space_size=100 \
  --aggressive-cache-discard \
  --start-maximized \
  http://localhost:8080

# If Chromium exits, restart it after 5 seconds
sleep 5
exec $0
