valuedomain-ddns
================

An _unofficial_ DDNS (Dynamic DNS) updater for [Value-Domain](https://www.value-domain.com/) written by Golang.

Usage
-----

```bash
./valuedomain-ddns -c ddns.json
```

Configuration
-------------

### ddns.json

Sample `/etc/ddns.json`:

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

Two output files (`checker.last` and `log.file`) are created automatically.


### cron

Sample `/etc/cron.d/valuedomain-ddns`:

```cron
*/15 * * * * USER /opt/mikan/valuedomain-ddns -c /etc/ddns.json > /dev/null
```

We do not recommend specifying root for _USER_, but you need to specify a user who has write access to log / last IP file.


Parameters
----------

* `-c` - path to configuration file (default "ddns.json")
* `-h` - print usage

Q & A
-----

##### How to find my password for DDNS?

See [ダイナミックDNSの設定方法と注意事項
](https://www.value-domain.com/ddns.php?action=howto).

DDNS password is different from Value-Domain login password.

##### How to update forcibly?

Remove last-IP file and re-run. The file path is configured at `checker.last`.

Author
-----

[mikan](https://github.com/mikan)

License
-------

[The 3-Clause BSD License](LICENSE)
