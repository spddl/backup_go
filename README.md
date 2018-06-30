# Backup_GO
Automatically schedule timed backups of directories.

### Config.json

```json
{
  "path": ".\\save\\",
  "interval": 60,
  "files": [{
    "name": "world123",
    "path": "D:\\git\\backup_go\\HelloWorld123",
    "except": [
      "HelloWorld123\\logs\\1.txt"
    ],
    "skipCRCCheck": false,
    "keepLastFiles": 3
  }]
}
```