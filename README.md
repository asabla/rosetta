# Rosetta Cedar + OpenShell Control Plane

Minimal service that accepts sandbox capability requests, authorizes them through a Cedar-facing authorizer, deterministically compiles granted capabilities into OpenShell policy YAML, and invokes OpenShell only through an adapter interface.

## Assumptions

OpenShell documents policy YAML fields, dynamic policy updates, and security constraints, but this wrapper does not assume an undocumented Go API. `internal/openshell.Adapter` is the single integration seam. The default adapter logs calls; production deployments can replace it with a CLI/gRPC/HTTP implementation if NVIDIA documents or deploys one.

## Security model

* Default deny in the Cedar authorizer.
* Agents submit typed capabilities only; raw OpenShell YAML is never accepted.
* Wildcard hosts are rejected and hostnames are normalized.
* Unsafe paths (`..`, symlinks, `/etc`, `/proc`, `/sys`, `/root`, likely host mounts) are rejected before grants.
* Long-lived secrets are rejected and secrets are not embedded in generated policy YAML.
* Every generated policy is written to `./generated-policies` and identified by a SHA-256 hash.
* Logs are structured JSON for Cedar decisions and OpenShell adapter calls.

## Run

```bash
go run ./cmd/rosetta
```

## API examples

Create a sandbox:

```bash
curl -sS -X POST localhost:8080/sandboxes \
  -H 'content-type: application/json' \
  -d '{"principal":"agent"}'
```

Dry-run sandbox creation:

```bash
curl -sS -X POST localhost:8080/sandboxes \
  -H 'content-type: application/json' \
  -d '{"principal":"agent","dry_run":true}'
```

Grant network capability:

```bash
curl -sS -X POST localhost:8080/sandboxes/$SANDBOX_ID/capabilities/network \
  -H 'content-type: application/json' \
  -d '{"principal":"agent","host":"api.github.com","port":443,"binary":"/usr/bin/curl"}'
```

Authorize a short-lived secret without embedding it in YAML:

```bash
curl -sS -X POST localhost:8080/sandboxes/$SANDBOX_ID/capabilities/secrets \
  -H 'content-type: application/json' \
  -d '{"principal":"agent","secret":"github-token","long_lived":false}'
```

Fetch generated OpenShell policy YAML:

```bash
curl -sS localhost:8080/sandboxes/$SANDBOX_ID/policy
```

## Cedar files

* `cedar/schema.cedarschema` declares entities, actions, resources, and request context for capability authorization.
* `cedar/policies/*.cedar` contains the checked-in permit rules used by the service.

## OpenShell compiler

The compiler emits schema version `1`, filesystem policy, hard Landlock requirement, non-root process identity, and deterministic network policy entries. It intentionally contains no tenant/business decisions; only already-authorized typed grants are converted to YAML.
