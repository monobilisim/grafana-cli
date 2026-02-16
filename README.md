# [![Contributors][contributors-shield]][contributors-url]

[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![GPL License][license-shield]][license-url]

[![Readme in English](https://img.shields.io/badge/Readme-English-blue)](README.md)

<div align="center">  
<a href="https://mono.net.tr/">  
  <img src="https://monobilisim.com.tr/images/mono-bilisim.svg" width="340"/>  
</a>

<h1 align="center">gcli - Grafana API CLI Tool</h1>

`gcli` is a powerful Go-based command-line interface for interacting with the Grafana API. It allows you to manage multiple Grafana profiles, switch between organizations, and manage data sources with ease.

</div>

---

## Implemented Features

- **Profile Management**: Save multiple Grafana instances and switch between them.
- **Organization (Tenant) Management**: List, create, delete, update, and switch active organizations.
- **Data Source Management**: Full CRUD operations for data sources (List, Create, Read details, Update, Delete) with tabular output and interactive editing.
- **Dashboard Management**: List, read, create (handles external templates with mapping), update (interactive editor), and delete dashboards.
- **Generic Requests**: Make raw API calls to any Grafana endpoint.

## Roadmap

Upcoming features and improvements:
- **Folder Management**: Full control over dashboard folders.
- **User Management**: Manage Grafana users via CLI.
- **Group/Team Management**: Manage teams and permissions.
- **Improved Output Formats**: More flexible output options (CSV, etc.).

## Installation

### Prerequisites
- Go 1.20 or later

### Build & Install
Clone the repository and run:
```bash
make install
```
This will generate the `gcli` binary, move it to `/usr/local/bin`, and set up auto-completion for your shell.

## Usage

### 1. Configuration (`gcli config`)
Manage your Grafana API profiles.

- **Add a profile**:
  ```bash
  ./gcli config add --name my-grafana --url https://grafana.example.com --user admin --pass secret
  ```
- **List profiles**:
  ```bash
  ./gcli config list
  ```
- **Set active profile**:
  ```bash
  ./gcli config use my-grafana
  ```

### 2. Organization Management (`gcli org`)
Manage organizations in the active Grafana profile.

- **List organizations**:
  ```bash
  ./gcli org list
  ```
- **Select active organization**:
  ```bash
  ./gcli org use "Main Org."
  ```
- **Create organization**:
  ```bash
  ./gcli org create --name "New Org"
  ```
- **Delete organization**:
  ```bash
  ./gcli org rm 12
  ```
- **Update organization**:
  ```bash
  ./gcli org update 12 --name "New Name"
  ```

### 3. Dashboard Management (`gcli dash`)
Manage your Grafana dashboards.

- **List dashboards**:
  ```bash
  ./gcli dash list
  ```
- **Read dashboard** (extracts `.dashboard` field):
  ```bash
  ./gcli dash read <uid>
  ```
- **Export for sharing** (external template):
  ```bash
  ./gcli dash read <uid> --external
  ```
- **Create dashboard** (handles external templates with datasource mapping):
  ```bash
  ./gcli dash create --file dash.json
  ```
- **Update dashboard** (interactive editor with retry logic):
  ```bash
  ./gcli dash update <uid>
  ```
- **Remove dashboard**:
  ```bash
  ./gcli dash rm <uid>
  ```

### 4. Data Source Management (`gcli ds`)
Manage data sources in the active organization.

- **List data sources** (tabular view):
  ```bash
  ./gcli ds list
  ```
- **List data sources** (full JSON details):
  ```bash
  ./gcli ds list --details
  ```
- **Read data source details**:
  ```bash
  ./gcli ds read "Prometheus"
  ```
- **Create data source** (via flags):
  ```bash
  ./gcli ds create --name my-db --type graphite --url http://localhost:8080 --access proxy
  ```
- **Create data source** (via file):
  ```bash
  ./gcli ds create --file ds.json
  ```
- **Update data source** (interactive editor):
  ```bash
  # Opens your default $EDITOR with the current config
  ./gcli ds update my-db
  ```
- **Update data source** (via file):
  ```bash
  ./gcli ds update my-db --file update.json
  ```
- **Remove data source**:
  ```bash
  ./gcli ds rm my-db
  ```

### 4. Generic Request (`gcli request`)
Make any request to the active Grafana instance.
```bash
./gcli request GET /api/health
```

## Configuration File
Profiles and active settings are stored in `~/.gcli/config.yaml`.

---

[contributors-shield]: https://img.shields.io/github/contributors/monobilisim/grafana-cli.svg?style=for-the-badge
[contributors-url]: https://github.com/monobilisim/grafana-cli/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/monobilisim/grafana-cli.svg?style=for-the-badge
[forks-url]: https://github.com/monobilisim/grafana-cli/network/members
[stars-shield]: https://img.shields.io/github/stars/monobilisim/grafana-cli.svg?style=for-the-badge
[stars-url]: https://github.com/monobilisim/grafana-cli/stargazers
[issues-shield]: https://img.shields.io/github/issues/monobilisim/grafana-cli.svg?style=for-the-badge
[issues-url]: https://github.com/monobilisim/grafana-cli/issues
[license-shield]: https://img.shields.io/github/license/monobilisim/grafana-cli.svg?style=for-the-badge
[license-url]: https://github.com/monobilisim/grafana-cli/blob/master/LICENSE
