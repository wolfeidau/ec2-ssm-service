# ec2-ssm-service

This is a cli command which which downloads configuration files, and env files from SSM and writes them to the local system.

# configuration

```
config:
    configfile:
        /etc/blah/blah.cfg: /dev/main/ci/blah-cfg
env:
    envfile:
        /etc/blah/blah.env: /dev/main/ci/blah
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

# License

This application is released under Apache 2.0 license and is copyright [Mark Wolfe](https://www.wolfe.id.au).