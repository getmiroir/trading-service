# Tests

This directory contains the repository's test suite.

## Structure

Tests are organized under `local/` for end-to-end testing that requires API credentials and runs on developer machines.

## Setup

Create a `.env.test` file in the project root with all required API keys for testing.

The `setup_test.go` file handles:
- Loading environment variables from `.env.test`
- Initializing test parameters
- Running all tests in the folder

For Binance create the API keys from:
- your Binance account in the demo window
- https://testnet.binance.vision/ to create the test spot API key

## Running Tests

**Run all local tests:**
```bash
go test -v -count=1 ./tests/local/
```

**Run a specific test:**
When running individual tests, you'll need to manually set test parameters in your code.