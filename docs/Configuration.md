# Configuration

`gcli` uses a configuration file located at `~/.gcli/config.yaml`.

## Adding a Profile

Add your first Grafana profile:
```bash
gcli config add --name prod --url https://grafana.example.com --user admin --pass secret
```

## Managing Profiles

- **List all profiles**: `gcli config list`
- **Switch active profile**: `gcli config use <name>`

## Organization Context

Most commands operate within the context of an active organization.
- **List organizations**: `gcli org list`
- **Switch organization**: `gcli org use <name-or-id>`
