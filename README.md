![Baton Logo](./baton-logo.png)

# `baton-freshdesk` [![Go Reference](https://pkg.go.dev/badge/github.com/conductorone/baton-freshdesk.svg)](https://pkg.go.dev/github.com/conductorone/baton-freshdesk) ![main ci](https://github.com/conductorone/baton-freshdesk/actions/workflows/main.yaml/badge.svg)

`baton-freshdesk` is a connector for Freshdesk built using the [Baton SDK](https://github.com/conductorone/baton-sdk). It communicates with the [Freshdesk API](https://developers.freshdesk.com/api/) to syncronize data from the platform and gather information about the Users. This connnector allows you to visualize the permits of each user (the roles they have) and to modify them by adding or removing roles.

Check out [Baton](https://github.com/conductorone/baton) to learn more the project in general.


# Prerequisites
In order to use this connector, you need to have an API Key to access the Freshdesk domain. Both the Token and the Domain must be indicated with the flags --api-key and --domain.
Example: 
  For connecting to https://example.freshdesk.com you should do:
  
  ```
  baton-freshdesk --api-key abcdefghij1234567890 --domain example
  ```

## Where can I find my API key?
    1. Log in to your Support Portal
    2. Click on your profile picture on the top right corner of your portal
    3. Go to Profile settings Page
    4. Your API key will be available below the change password section to your right


# Getting Started

## brew

```
brew install conductorone/baton/baton conductorone/baton/baton-freshdesk
baton-freshdesk
baton resources
```

## docker

```
docker run --rm -v $(pwd):/out -e BATON_DOMAIN_URL=domain_url -e BATON_API_KEY=apiKey -e BATON_USERNAME=username ghcr.io/conductorone/baton-freshdesk:latest -f "/out/sync.c1z"
docker run --rm -v $(pwd):/out ghcr.io/conductorone/baton:latest -f "/out/sync.c1z" resources
```

## source

```
go install github.com/conductorone/baton/cmd/baton@main
go install github.com/conductorone/baton-freshdesk/cmd/baton-freshdesk@main

baton-freshdesk

baton resources
```

# Data Model

`baton-freshdesk` will pull down information about the following resources:
- Users
- Roles
- Groups

# Contributing, Support and Issues

We started Baton because we were tired of taking screenshots and manually
building spreadsheets. We welcome contributions, and ideas, no matter how
small&mdash;our goal is to make identity and permissions sprawl less painful for
everyone. If you have questions, problems, or ideas: Please open a GitHub Issue!

See [CONTRIBUTING.md](https://github.com/ConductorOne/baton/blob/main/CONTRIBUTING.md) for more details.

# `baton-freshdesk` Command Line Usage

```
baton-freshdesk

Usage:
  baton-freshdesk [flags]
  baton-freshdesk [command]

Available Commands:
  capabilities       Get connector capabilities
  completion         Generate the autocompletion script for the specified shell
  help               Help about any command

Flags:
      --client-id string             The client ID used to authenticate with ConductorOne ($BATON_CLIENT_ID)
      --client-secret string         The client secret used to authenticate with ConductorOne ($BATON_CLIENT_SECRET)
  -f, --file string                  The path to the c1z file to sync with ($BATON_FILE) (default "sync.c1z")
  -h, --help                         help for baton-freshdesk
      --log-format string            The output format for logs: json, console ($BATON_LOG_FORMAT) (default "json")
      --log-level string             The log level: debug, info, warn, error ($BATON_LOG_LEVEL) (default "info")
  -p, --provisioning                 If this connector supports provisioning, this must be set in order for provisioning actions to be enabled ($BATON_PROVISIONING)
      --ticketing                    This must be set to enable ticketing support ($BATON_TICKETING)
  -v, --version                      version for baton-freshdesk

Use "baton-freshdesk [command] --help" for more information about a command.
```
