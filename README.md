# ec2-ssm-service

This is a cli command which which downloads configuration files, and env files from SSM and writes them to the local system.

# configuration

```
configs:
    /dev/main/ci/blah/config/blah-cfg: /etc/blah/blah.cfg
env-files:
    /dev/main/ci/blah/env: /etc/blah/blah.env
```

# Usage

Here's how to configure a systemd service to depend on ec2-ssm-config.service and ec2-ssm-env.service:

```
[Unit]
Description=Example Dependent Service
After=ec2-ssm-config.service
Requires=ec2-ssm-config.service
# or alternatively use Wants= for a softer dependency
After=ec2-ssm-env.service
Requires=ec2-ssm-env.service
# or alternatively use Wants= for a softer dependency

[Service]
Type=simple
ExecStart=/path/to/your/service
# other service configuration...

[Install]
WantedBy=multi-user.target
```

The key directives are:

1. `After=ec2-ssm-config.service` - This ensures that the dependent service starts after ec2-ssm-config.service has started.

2. `Requires=ec2-ssm-config.service` - This creates a strong dependency. If ec2-ssm-config.service fails, this service won't start. If this service is started, ec2-ssm-config.service will be started first if it's not already running.

Since ec2-ssm-config.service is configured as `Type=oneshot` with `RemainAfterExit=yes`, `systemd` will consider it "started" only after the command has completed successfully, ensuring your dependent service starts only after the configuration has been fully applied to the system.

# Install

```bash
curl -L https://github.com/wolfeidau/ec2-ssm-service/releases/download/v0.1.0/ec2-ssm-service-v0.1.0-linux-arm64.deb -o /tmp/ec2-ssm-service-v0.1.0-linux-arm64.deb
dpkg -i /tmp/ec2-ssm-service-v0.1.0-linux-arm64.deb
```

# License

This application is released under Apache 2.0 license and is copyright [Mark Wolfe](https://www.wolfe.id.au).