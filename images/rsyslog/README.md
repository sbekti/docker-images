# rsyslog

Minimal non-root rsyslog runtime for amd64 and arm64. Mount configuration at
`/etc/rsyslog.conf` and provide writable volumes for its configured output and
working directories. UID and GID are 10001. No listener, destination, retention
policy, or credentials are configured by this image.

Build with `docker build -t rsyslog:test images/rsyslog` and run
`sh images/rsyslog/test.sh rsyslog:test`.

The base is Alpine 3.24.4 or newer, from the release containing the SSHole fix.
OpenSSH is not installed. Runtime package updates are included during builds.
