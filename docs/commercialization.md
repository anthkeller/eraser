# Commercialization and deployment model

Eraser is designed as a single-user privacy product with two deployment modes.
The same Go application and versioned API should serve both modes so security
fixes and broker workflows do not split into separate codebases.

## Product editions

| Edition | Deployment | Intended use |
| --- | --- | --- |
| Community | Self-hosted | One person running Eraser on hardware they control |
| Cloud | Hosted | The official subscription service |
| Commercial | Self-hosted | Licensed internal or partner deployment |

Every edition is single-user today. The internal `owner_id` setting reserves the
identity that future storage and job interfaces will use, without exposing
family roles or an RBAC interface. It does not yet enforce database-level tenant
filtering. Hosted mode must not
be enabled until authentication and owner-scoped storage are implemented.

## Runtime configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `ERASER_EDITION` | `community` | `community`, `cloud`, or `commercial` |
| `ERASER_DEPLOYMENT` | `self-hosted` | `self-hosted` or `hosted` |
| `ERASER_BIND` | `127.0.0.1` | Network address accepted by the HTTP server |
| `ERASER_PUBLIC_URL` | empty | Canonical external HTTP or HTTPS URL |
| `ERASER_OPEN_BROWSER` | `true` | Open the local dashboard at startup |
| `ERASER_OWNER_ID` | `local` | Internal owner boundary; never returned publicly |
| `ERASER_AUTH_ISSUER` | empty | Hosted OIDC issuer URL |
| `ERASER_AUTH_AUDIENCE` | empty | Hosted OIDC client ID or token audience |

Hosted mode rejects the community edition and rejects an HTTP public URL. When
the public URL uses HTTPS, Eraser marks CSRF cookies secure and trusts only that
URL's host in addition to local development origins.

Hosted mode also requires a non-default owner ID and a discoverable OIDC
provider. Every route except health, readiness, and public system metadata
requires a verified bearer token. Its `sub` claim must exactly match
`ERASER_OWNER_ID`. This creates a strong single-subscriber deployment boundary
without presenting family roles or RBAC.

Operational endpoints:

- `GET /healthz` reports whether the process is serving HTTP.
- `GET /readyz` verifies that the application database responds.
- `GET /api/v1/system` returns non-secret edition, version, deployment, and
  capability metadata for web and mobile clients.
- `GET /api/v1/me` returns the authenticated subject and is protected in hosted
  mode.

## Hosted target architecture

The current foundation includes OIDC verification and a queue interface with an
owner ID on every job. The in-memory implementation supports self-hosted
operation. A future SQS implementation can satisfy the same interface.

The next shared-service milestone should add these replaceable interfaces:

1. PostgreSQL storage with an owner ID on every user-controlled record.
2. An SQS queue implementation for scan, validation, email, and recurring jobs.
3. Object storage for encrypted evidence with short retention periods.
4. Provider interfaces for email, notifications, billing, and entitlements.
5. An OIDC authorization-code flow for the browser UI. The current hosted
   interface expects a bearer token supplied by a mobile app, SPA, or gateway.

An AWS deployment can map these interfaces to Cognito, RDS PostgreSQL, SQS,
S3/KMS, EventBridge, SES, Secrets Manager, WAF, and ECS Fargate. Self-hosting can
continue to use local configuration, SQLite, the filesystem, and an in-process
worker.

## Licensing status

The repository currently describes itself as MIT licensed, has no standalone
license file, and contains commits from multiple authors. Existing MIT grants
cannot be withdrawn from copies already distributed. Do not replace the license
until ownership of all relevant code is documented and contributors have
provided any permissions required for relicensing.

The intended future policy is:

- personal self-hosting remains permitted;
- the official hosted service can be commercialized;
- third parties cannot offer substantially the same product as a hosted or
  managed service without a commercial agreement; and
- project trademarks and official update services remain separately controlled.

That policy requires a source-available or proprietary license rather than an
OSI-approved open-source license. Legal counsel should review the contributor
history, proposed Elastic License 2.0 terms, trademark policy, privacy policy,
terms of service, and contributor agreement before a relicensing release.
