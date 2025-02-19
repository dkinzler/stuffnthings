A CLI tool to backup your stuff.

## Usage

Run as an automated script:

```shell
backup --config config.json
```

Use terminal user interface:

```shell
backup --config config.json tui
```

### Configuration Example

For all options see [internal/config/config.go](internal/config/config.go).

```json
{
    "backupDir": "~/backup",
    "github": {
        "token": "your-personal-access-token-here"
    },
    "zip": {
        "file": "~/backup.zip"
    },
    "files": [
        "~/.config",
        "~/Downloads/abc.zip",
        "/abc/def"
    ]
}
```
