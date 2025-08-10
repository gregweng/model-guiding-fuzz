#!/bin/bash

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to cleanup background processes on script exit
cleanup() {
    echo "Cleaning up..."
    if [ ! -z "$EMULATOR_PID" ]; then
        kill $EMULATOR_PID 2>/dev/null
    fi
    exit
}

# Set up cleanup trap
trap cleanup EXIT INT TERM

# Check for required commands
if ! command_exists gcloud; then
    echo "Error: gcloud is not installed. Installing..."
    if command_exists brew; then
        brew install google-cloud-sdk
    else
        echo "Error: Homebrew is not installed. Please install gcloud manually:"
        echo "Visit: https://cloud.google.com/sdk/docs/install"
        exit 1
    fi
fi

# Initialize gcloud if needed
if [ ! -f "$HOME/.config/gcloud/configurations/config_default" ]; then
    echo "Initializing gcloud..."
    gcloud init --console-only --skip-diagnostics --quiet || {
        echo "Failed to initialize gcloud. Please run 'gcloud init' manually first."
        exit 1
    }
fi

# Accept terms of service non-interactively
echo "Accepting terms of service..."
yes | gcloud beta emulators pubsub start --help > /dev/null 2>&1 || true

# Install pubsub emulator if not already installed
if ! gcloud components list --installed --quiet | grep -q "pubsub-emulator"; then
    echo "Installing pubsub emulator..."
    gcloud components install pubsub-emulator --quiet
fi

# Create a temporary file for emulator output
EMULATOR_LOG=$(mktemp)

# Check if port 8085 is already in use
if lsof -i :8085 > /dev/null 2>&1; then
    echo "Error: Port 8085 is already in use. Please free up the port and try again."
    echo "You can use 'lsof -i :8085' to find which process is using it."
    exit 1
fi

# Start the emulator in the background with more detailed output
echo "Starting Pub/Sub emulator..."
gcloud beta emulators pubsub start \
    --project=test-project \
    --host-port=localhost:8085 \
    --quiet \
    > "$EMULATOR_LOG" 2>&1 &
EMULATOR_PID=$!

# Wait for emulator to start by checking the log file
MAX_RETRIES=30
RETRY_COUNT=0
while ! grep -q "Server started" "$EMULATOR_LOG" 2>/dev/null; do
    if ! kill -0 $EMULATOR_PID 2>/dev/null; then
        echo "Error: Emulator failed to start. Error details:"
        echo "----------------------------------------"
        cat "$EMULATOR_LOG"
        echo "----------------------------------------"
        exit 1
    fi
    
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
        echo "Error: Timeout waiting for emulator to start. Last few lines of log:"
        echo "----------------------------------------"
        tail -n 10 "$EMULATOR_LOG"
        echo "----------------------------------------"
        kill $EMULATOR_PID 2>/dev/null
        exit 1
    fi
    
    echo "Waiting for emulator to start... (attempt $RETRY_COUNT/$MAX_RETRIES)"
    sleep 1
done

# Verify the emulator is actually responding
echo "Verifying emulator is responsive..."
if ! curl -s localhost:8085 > /dev/null 2>&1; then
    echo "Error: Emulator is not responding on port 8085. Last few lines of log:"
    echo "----------------------------------------"
    tail -n 10 "$EMULATOR_LOG"
    echo "----------------------------------------"
    kill $EMULATOR_PID 2>/dev/null
    exit 1
fi

# Set the emulator host environment variable
export PUBSUB_EMULATOR_HOST=localhost:8085
echo "Emulator is running at $PUBSUB_EMULATOR_HOST"

# Build and run the example
echo "Running the example..."
cd "$(dirname "$0")"
go run main.go

# Keep the script running to maintain the emulator
echo -e "\nEmulator is still running. Press Ctrl+C to stop."
wait $EMULATOR_PID
