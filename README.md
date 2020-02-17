# godaddyupdater

Godaddy dns auto updater. Checks if your public ip have changed, and updates godaddy dns record if changed.

## Example

```bash
go run main.go -auth="flag" -key="mySecretKey" -secret="mySecretSecret" -checkInterval=60 -domain="erter.org" -subDomain="dev"
2020/02/17 20:43:20 Current godaddy ip for dev.erter.org = 193.69.46.161
2020/02/17 20:44:20 My IP is:193.69.46.162
2020/02/17 20:44:20 * The ip's are different, preparing to update record at godaddy.
2020/02/17 20:44:21 Updating godaddy DNS record, OK
2020/02/17 20:44:21 The ip address have not changed, keeping everything as it is.
2020/02/17 20:45:21 My IP is:193.69.46.162
2020/02/17 20:45:21 The ip address have not changed, keeping everything as it is.
```

NB: Godaddy don't allow faster use of an API endpoint than 60 seconds.
