# Usage Examples

## Dashboard Management

### Exporting a Dashboard
```bash
gcli dash read <uid> --external > dashboard-template.json
```

### Importing a Dashboard
```bash
gcli dash create --file dashboard-template.json
```

### Interactive Edit
```bash
gcli dash update <uid>
```

## Data Source Management

### Listing Data Sources
```bash
gcli ds list
```

### Creating from JSON
```bash
gcli ds create --file datasource.json
```

## Generic API Requests
```bash
gcli request GET /api/admin/settings
```
