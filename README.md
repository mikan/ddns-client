valuedomain-ddns
================

An _unofficial_ DDNS (Dynamic DNS) updater for [Value-Domain](https://www.value-domain.com/) and [Cloudflare](https://www.cloudflare.com/) written by Golang.

Download
--------

### Binary distribution

Download from [releases page](https://github.com/mikan/valuedomain-ddns/releases). Windows and Linux (amd64) are available.

### go get

```bash
go get github.com/mikan/ddns-client
```

Usage
-----

### One-shot

```bash
./valuedomain-ddns -c ddns.json
```

### cron

Sample `/etc/cron.d/ddns`:

```cron
*/15 * * * * USER /opt/mikan/ddns -c /etc/ddns.json > /dev/null
```

We do not recommend specifying root for _USER_, but you need to specify a user who has write access to log / last IP file.

Configuration
-------------

### ddns.json

Sample `/etc/ddns.json` for Value-Domain:

```json
{
  "targets": [
    {
      "class": "valuedomain",
      "domain": "foo.com",
      "password": "DDNS-PASSWORD",
      "host": "*"
    }
  ],
  "checker": {
    "method": "web",
    "url": "https://dyn.value-domain.com/cgi-bin/dyn.fcg?ip",
    "last": "/var/log/ddns.last"
  },
  "log": {
    "file": "/var/log/ddns.log"
  }
}
```

Sample `/etc/ddns.json` for Cloudflare:

```json
{
  "targets": [
    {
      "class": "cloudflare",
      "domain": "ZONE-ID",
      "password": "API-KEY",
      "host": "host.foo.com"
    }
  ],
  "checker": {
    "method": "web",
    "url": "https://checkip.amazonaws.com/",
    "last": "/var/log/ddns.last"
  },
  "log": {
    "file": "/var/log/ddns.log"
  }
}
```

Two output files (`checker.last` and `log.file`) are created automatically.

Parameters
----------

* `-c` - path to configuration file (default "ddns.json")
* `-h` - print usage

Q & A
-----

##### How to find my password for Value-Domain DDNS?

See [ダイナミックDNSの設定方法と注意事項
](https://www.value-domain.com/ddns.php?action=howto).

DDNS password is different from Value-Domain login password.

##### How to update forcibly?

Remove last-IP file and re-run. The file path is configured at `checker.last`.

##### What are the supported IP checkers?

- Value-Domain: `https://dyn.value-domain.com/cgi-bin/dyn.fcg?ip`
- ipify: `https://api.ipify.org/`
- AWS: `https://checkip.amazonaws.com/` (supported since v0.2)

Author
-----

[mikan](https://github.com/mikan)

License
-------

[The 3-Clause BSD License](LICENSE)
