# godaddyupdater

Godaddy dns auto updater. Checks if your public ip have changed, and updates godaddy dns record if changed.

## About

* Supports key and secret for godaddy API given as env variables, or provided via flags.
* Built in run scheduler so no need for Cron. Interval set via flag value.
* Can update both main domain, or sub domain records.
* Prometheus Exporter for exporting time series data of line status.

## flags

Default is to use env variables for both godaddy key and secret.

``` bash
  -auth string
    Use "env" or "flag" for way to get key and secret.\n
    if value chosen is "flag", use the -key and -secret flags.\
    if value chosen is "env", set the env variables "goddaddykey" and "godaddysecret"
     (default "env")
  -checkInterval int
    check interval in seconds (default 5)
  -domain string
    domain name, e.g. -domain="erter.org. NB: If you want to update the main domain like erter.org use "@" as value with the subDomain flag like  -subDomain="@""
  -key string
    the key you got at https://developer.godaddy.com/keys
  -secret string
    the secret you got at https://developer.godaddy.com/keys
  -subDomain string
    domain name, e.g. -subDomain="dev". NB: If you want to update the main domain like erter.org use "@" as value like -subDomain="@"
  -promExpPort string
    The port number to run the prometheus exporter on written as a string value. Default : -promExpPort="2112" (default "2112")
```

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

## Dockerize it

### Create an image

Change the startup options like Domain, Prometheus listen port, or scheduled intervals, edit the appropriate settings in the `Dockerfile` before building.
Current defaults are :
`CMD ["/app/main", "-auth=env" ,"-checkInterval=99" ,"-domain=erter.org" ,"-subDomain=dev" ,"-promExpPort=9101"]`

From the repository folder:
`docker image build -t godaddyupdater:1.0 .`

### Run the container

Create a file for the environment variables, for example `env.env` like this:

```text
godaddykey=supersecretkey
godaddysecret=supersecretsecret
```

Then run the docker container:
`docker run -it --env-file env.env -p 9101:9101 godaddyupdater:1.0`

To make it start in daemon mode:
`docker run -d --env-file env.env -p 9101:9101 godaddyupdater:1.0`

