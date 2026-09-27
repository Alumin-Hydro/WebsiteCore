# Mac mini CD

`macmini-cd-controller.sh` is a private pull controller. It reads only
`refs/heads/main` from the public GitHub repository, and deploys a new SHA when
that SHA differs from `runtime/deployed_sha`. It never accepts GitHub Actions
jobs or pull-request code, so the public repository cannot use the Mac mini's
Docker socket through a self-hosted runner.

The controller is itself a Docker container with the Docker Desktop socket
mounted. The application stack remains private on `127.0.0.1:18008`; the
existing `gsc-wanderin` Cloudflare Tunnel routes `demo.wanderin.cn` to the app
through `gsc-net`.

## One-time bootstrap

The runtime directory is private and must not be committed:

- host root: `/Users/alumin/Project/websitecore-cd`
- runtime config: `runtime/custom/config.yaml` (0600)
- PostgreSQL/Redis/Meilisearch data: `runtime/{postgres,redis,meili}`
- deploy backups: `runtime/backups`
- checked-out source: `source`

Build and start the controller on the Mac mini:

```sh
cd /Users/alumin/Project/websitecore-ci-review
mkdir -p /Users/alumin/Project/websitecore-cd/source

docker build \
  --build-arg HTTP_PROXY=http://host.docker.internal:7890 \
  --build-arg HTTPS_PROXY=http://host.docker.internal:7890 \
  -f deploy/macmini-cd-controller.Dockerfile \
  -t websitecore-macmini-cd:latest deploy

docker run -d \
  --name websitecore-macmini-cd \
  --restart unless-stopped \
  -e MACMINI_DEPLOY_ROOT=/Users/alumin/Project/websitecore-cd \
  -e RUNTIME_DIR=/opt/deploy/runtime \
  -e POLL_SECONDS=60 \
  -e HTTP_PROXY=http://host.docker.internal:7890 \
  -e HTTPS_PROXY=http://host.docker.internal:7890 \
  -e http_proxy=http://host.docker.internal:7890 \
  -e https_proxy=http://host.docker.internal:7890 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /Users/alumin/Project/websitecore-cd:/opt/deploy \
  -v /Users/alumin/Project/websitecore-cd/source:/opt/source \
  websitecore-macmini-cd:latest
```

The controller initially deploys the current `main`. Every later merge to
`main` is picked up within one polling interval. `docker logs -f
websitecore-macmini-cd` shows the SHA and deployment result.

## Runtime/data policy

WebsiteCore starts with an empty PostgreSQL database. The downloaded Starisle
backup is deliberately not restored: it belongs to a different Node.js schema.
The deploy script runs WebsiteCore migrations, keeps the database/Redis/Meili
volumes across app rebuilds, and creates a compressed PostgreSQL backup before
replacing an existing app container. A failed app health/API check attempts to
restore the previous immutable image; database schema rollback is not assumed
and the backup path is retained for manual recovery.
