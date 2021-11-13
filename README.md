Simplest possible Dynamic DNS client & server with acceptable security.


Warning
-------

This is not intended as ready-to-run product for anybody to use, but rather as a reference / learning subject.

There are currently some hardcoded bits and one file missing (which maps hostname to cloudflare zone + record IDs)


Server
------

Runs on Lambda to makes changes to hostname's DNS record if client asks to.

Basically implements `PUT /dyndns/api/hostname/<hostname>`

Currently assumes you're using Cloudflare as your DNS service.


### Quickstart

- Generate new update token validator secret: `$ dyndns server update-token-validator-secret-generate`.
  Set this as ENV var `UPDATE_TOKEN_VALIDATOR_SECRET`.
- Create API Token in Cloudflare and set this as ENV var `CLOUDFLARE_API_TOKEN`.
  Scope the token to only the zones that require dynamic DNS.


Client
------

Runs as a scheduled task (e.g. every 5 min). Knows its own hostname. Queries:

- DNS for hostname's IP
- Remote IP service to resolve our own public IP

If these don't match, asks server to update the DNS record.


### Quickstart

Run `$ dyndns server update-token-gen <hostname>` to get update auth token for hostname.

(The auth token is scoped to the hostname, so if an attacker manages to steal it, they can't make
changes to other DNS records.)

Then hook up `$ dyndns client <hostname> <hostnameAuthToken>` to your client machine's scheduler.

