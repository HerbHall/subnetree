---
name: Release Verification
about: Track build verification across platforms for a release
title: 'verify: vX.Y.Z release verification'
labels: 'release, testing'
assignees: ''

---

## Release Verification: vX.Y.Z

**Release:** [vX.Y.Z](https://github.com/HerbHall/subnetree/releases/tag/vX.Y.Z)
**Release Date:** YYYY-MM-DD

### Binary Verification

#### Windows

- [ ] **Windows amd64** -- `subnetree_X.Y.Z_windows_amd64.zip`
  - [ ] Binary extracts and runs
  - [ ] `subnetree.exe -version` shows correct version, commit, and build date
  - [ ] Server starts with config file
  - [ ] `/healthz` returns 200
  - [ ] Dashboard loads at `/`
  - [ ] Setup wizard completes (fresh data dir)
  - Platform: <!-- e.g., Windows 11 23H2 -->
  - Notes:

- [ ] **Windows arm64** -- `subnetree_X.Y.Z_windows_arm64.zip`
  - [ ] Binary extracts and runs
  - [ ] `subnetree.exe -version` shows correct version
  - Platform:
  - Notes:

#### Linux

- [ ] **Linux amd64** -- `subnetree_X.Y.Z_linux_amd64.tar.gz`
  - [ ] Binary extracts and runs
  - [ ] `./subnetree -version` shows correct version, commit, and build date
  - [ ] Server starts and serves dashboard
  - [ ] `/healthz` returns 200
  - [ ] Network scan discovers devices (requires NET_RAW capability)
  - Platform: <!-- e.g., Ubuntu 24.04, Debian 12, UnRAID 7.x -->
  - Notes:

- [ ] **Linux arm64** -- `subnetree_X.Y.Z_linux_arm64.tar.gz`
  - [ ] Binary extracts and runs
  - [ ] `./subnetree -version` shows correct version
  - Platform: <!-- e.g., Raspberry Pi OS, Ubuntu arm64 -->
  - Notes:

#### macOS

- [ ] **macOS amd64** -- `subnetree_X.Y.Z_darwin_amd64.tar.gz`
  - [ ] Binary extracts and runs
  - [ ] `./subnetree -version` shows correct version
  - Platform:
  - Notes:

- [ ] **macOS arm64** -- `subnetree_X.Y.Z_darwin_arm64.tar.gz`
  - [ ] Binary extracts and runs
  - [ ] `./subnetree -version` shows correct version
  - Platform:
  - Notes:

### Scout Agent Verification

- [ ] **scout_X.Y.Z_windows_amd64.zip** -- extracts, shows version
- [ ] **scout_X.Y.Z_linux_amd64.tar.gz** -- extracts, shows version
- [ ] **scout_X.Y.Z_linux_arm64.tar.gz** -- extracts, shows version
- [ ] **scout_X.Y.Z_darwin_amd64.tar.gz** -- extracts, shows version
- [ ] **scout_X.Y.Z_darwin_arm64.tar.gz** -- extracts, shows version

### Docker Verification

- [ ] **Docker amd64** -- `ghcr.io/herbhall/subnetree:vX.Y.Z`
  - [ ] `docker pull` succeeds
  - [ ] Container starts and passes health check
  - [ ] Dashboard accessible at mapped port
  - [ ] Setup wizard completes
  - [ ] Network scan works (with `NET_RAW` + `NET_ADMIN`)
  - Host OS:
  - Docker version:
  - Notes:

- [ ] **Docker arm64** -- `ghcr.io/herbhall/subnetree:vX.Y.Z`
  - [ ] `docker pull` succeeds
  - [ ] Container starts and passes health check
  - Host OS:
  - Notes:

- [ ] **Docker Compose**
  - [ ] `docker-compose up -d` with README example works
  - [ ] Container healthy after 60s
  - Notes:

### Functional Verification (pick one platform)

- [ ] **First-run setup** -- wizard completes, admin account created
- [ ] **Authentication** -- login/logout works, JWT refresh works
- [ ] **Network scan** -- discovers at least one device on local network
- [ ] **Device list** -- shows discovered devices with manufacturer info
- [ ] **Device detail** -- clicking a device shows full detail page
- [ ] **Topology** -- network map renders with discovered devices
- [ ] **Dark mode** -- toggle works, persists across refresh
- [ ] **Keyboard shortcuts** -- `?` opens shortcut help
- [ ] **Backup/restore** -- `subnetree backup` creates archive, restore works
- [ ] **Pulse monitoring** -- ICMP checks auto-created for discovered devices
- [ ] **Vault** -- can create and retrieve a credential (requires unseal)
- [ ] **Health endpoints** -- `/healthz`, `/readyz`, `/api/v1/health` all respond

### Release Artifacts

- [ ] **Checksums** -- `checksums.txt` present and matches downloaded files
- [ ] **SBOMs** -- `.sbom.json` files present for all archives
- [ ] **Release notes** -- changelog is accurate and complete
- [ ] **Docker tags** -- `vX.Y.Z` and `latest` tags both work

### Test Commands

<details>
<summary>Quick verification commands</summary>

```bash
# Binary (Linux/macOS)
tar xzf subnetree_X.Y.Z_linux_amd64.tar.gz
./subnetree -version
./subnetree -config config.yaml &
sleep 3
curl -s http://localhost:8080/healthz
curl -s http://localhost:8080/api/v1/setup/status

# Docker
docker run -d --name subnetree-test \
  -p 8080:8080 \
  -v subnetree-test:/data \
  --cap-add NET_RAW --cap-add NET_ADMIN \
  ghcr.io/herbhall/subnetree:vX.Y.Z
sleep 10
curl -s http://localhost:8080/healthz
docker logs subnetree-test --tail 20

# Cleanup
docker stop subnetree-test && docker rm subnetree-test
docker volume rm subnetree-test
```

</details>
