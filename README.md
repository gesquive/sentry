# sentry

An URL monitoring alerting service.

## Installing

### Compile
This project requires go1.8+ to compile. Just run `go get -u github.com/gesquive/sentry` and the executable should be built for you automatically in your `$GOPATH`.

Optionally you can run `make install` to build and copy the executable to `/usr/local/bin/` with correct permissions.

### Download
Alternately, you can download the latest release for your platform from [github](https://github.com/gesquive/sentry/releases).

Once you have an executable, make sure to copy it somewhere on your path like `/usr/local/bin` or `C:/Program Files/`.
If on a \*nix/mac system, make sure to run `chmod +x /path/to/sentry`.

## Configuration

### Precedence Order
The application looks for variables in the following order:
 - command line flag
 - environment variable
 - config file variable
 - default

So any variable specified on the command line would override values set in the environment or config file.

### Config File
The application looks for a configuration file at the following locations in order:
 - `./config.yml`
 - `~/.config/sentry/config.yml`
 - `/etc/sentry/config.yml`

Copy `config.example.yml` to one of these locations and populate the values with your own. Since the config could contain your SMTP username/password, make sure to set permissions on the config file appropriately so others cannot read it. A good suggestion is `chmod 600 /path/to/config.yml`.

If you are planning to run this app as a service/cronjob, it is recommended that you place the config in `/etc/sentry/config.yml`. Otherwise, if running from the command line, place the config in `~/.config/sentry/config.yml` and make sure to set `run_once: true`.

### Environment Variables
Optionally, instead of using a config file you can specify config entries as environment variables. Use the prefix "SENTRY_" in front of the uppercased variable name. For example, the config variable `log_file` would be the environment variable `SENTRY_LOG_FILE`.

## Usage

```console
Watches http/s URLs for unexpected responses.

Usage:
  sentry [flags]

Flags:
      --check                  Check the config for errors and exit
      --config string          Path to a specific config file (default "./config.yml")
  -l, --log-file string        Path to log file (default "/var/log/sentry.log")
  -n, --no-alerts              Disable all outgoing email alerts, log alerts only
      --run-once               Run once and print out target status
  -w, --smtp-password string   Authenticate the SMTP server with this password
  -o, --smtp-port uint32       The port to use for the SMTP server (default 25)
  -x, --smtp-server string     The SMTP server to send email through (default "localhost")
  -u, --smtp-username string   Authenticate the SMTP server with this user
  -v, --verbose                Print logs to stdout instead of file
      --version                Display the version number and exit
```

It is helpful to use the `--run-once` combined with the `--verbose` flags when first setting up to find any misconfigurations.

Optionally, a hidden debug flag is available in case you need additional output.
```console
Hidden Flags:
  -D, --debug                  Include debug statements in log output
```


### Cronjob
To run as a cronjob on an \*nix system create a cronjob entry under the user the app is run with. If running as root, you can copy `services/sentry.cron` to `/etc/cron.d/sentry` or copy the following into you preferred crontab:
```shell
  0  *  *  *  * /usr/local/bin/sentry --run-once
```

Add any flags/env vars needed to make sure the job runs as intended. If not using arguments, then make sure the config file path is specified with a flag or can be found at one of the expected locations.

### Service
By default, the process is setup to run as a service. Feel free to use upstart, init, runit or any other service manager to run the `sentry` executable.

Example systemd & upstart scripts can be found in the `services` directory.

## Documentation

This documentation can be found at github.com/gesquive/sentry

## License

This package is made available under an MIT-style license. See LICENSE.

## Contributing

PRs are always welcome!
