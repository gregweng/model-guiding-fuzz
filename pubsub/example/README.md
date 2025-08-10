# PubSub Emulator Example

This example demonstrates how to use the PubSub client with a local Google Cloud Pub/Sub emulator.

## Prerequisites

1. Install Google Cloud SDK:
   ```bash
   # For macOS with Homebrew
   brew install google-cloud-sdk

   # For other platforms, follow instructions at:
   # https://cloud.google.com/sdk/docs/install
   ```

2. Install the Pub/Sub emulator:
   ```bash
   gcloud components install pubsub-emulator
   ```

## Running the Example

### Option 1: Using the Automated Script

The easiest way to run the example is to use the provided script:

```bash
./run_example.sh
```

This script will:
1. Check for and install required dependencies (gcloud, pubsub-emulator)
2. Start the Pub/Sub emulator
3. Set up the environment
4. Run the example
5. Clean up automatically when you press Ctrl+C

### Option 2: Manual Setup

If you prefer to run things manually:

1. Start the Pub/Sub emulator in a terminal:
   ```bash
   gcloud beta emulators pubsub start --project=test-project
   ```

2. In another terminal, set the environment variable to point to the emulator:
   ```bash
   export PUBSUB_EMULATOR_HOST=localhost:8085
   ```

3. Run the example:
   ```bash
   go run main.go
   ```

## Expected Output

If everything works correctly, you should see output similar to:
```
Published message with ID: <message-id>
Received message: Hello, PubSub Emulator!
Message attributes: map[sender:example time:2024-03-21T10:30:00Z]
```

## Troubleshooting

1. If you get connection errors, make sure:
   - The emulator is running
   - The `PUBSUB_EMULATOR_HOST` environment variable is set correctly
   - No firewall is blocking localhost:8085

2. If messages aren't received:
   - Check that the topic and subscription were created successfully
   - Verify that the publish operation succeeded
   - Try increasing the receive timeout in the code

## Cleanup

To stop the emulator, press Ctrl+C in the terminal where it's running.
