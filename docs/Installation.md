# Installation

## Prerequisites

- [Go](https://go.dev/dl/) 1.20 or later

## Building and Installing

### Method 1: Using Makefile (Recommended)

1. Clone the repository:
   ```bash
   git clone https://github.com/your-username/grafana-cli.git
   cd grafana-cli
   ```

2. Install the binary and auto-completion:
   ```bash
   make install
   ```
   This will move `gcli` to `/usr/local/bin` and configure your shell profile for auto-completion.

### Method 2: Manual Build

1. Build the binary:
   ```bash
   make build
   ```

2. Move it to your path:
   ```bash
   sudo mv gcli /usr/local/bin/
   ```

3. Setup completion manually:
   ```bash
   gcli completion install
   ```

## Verification

Run the following command to verify the installation and see available commands:
```bash
gcli --help
```

### Auto-completion

If you installed via `make install` or ran `gcli completion install`, auto-completion should work after restarting your shell. You can test it by typing `gcli ` and pressing `Tab`.
