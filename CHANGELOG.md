# Changelog

## [0.6.4](https://github.com/HerbHall/subnetree/compare/v0.6.3...v0.6.4) (2026-04-23)


### Features

* Ansible dynamic inventory plugin and YAML export endpoint ([#580](https://github.com/HerbHall/subnetree/issues/580)) ([7f41f77](https://github.com/HerbHall/subnetree/commit/7f41f778d939ecc01a11fc20a416be8f54d92f9a))


### Bug Fixes

* add workflow_dispatch to release workflow ([#577](https://github.com/HerbHall/subnetree/issues/577)) ([10f1bcd](https://github.com/HerbHall/subnetree/commit/10f1bcde8478274fc5beddfd4cbae92f3e6588f7))
* auto-fit topology viewport to readable node size ([#578](https://github.com/HerbHall/subnetree/issues/578)) ([0aa6f50](https://github.com/HerbHall/subnetree/commit/0aa6f5084fc16a460b7746fc1caf11dd1bc9204b))
* **deps:** resolve 6 Dependabot alerts in web deps ([#581](https://github.com/HerbHall/subnetree/issues/581)) ([47a4729](https://github.com/HerbHall/subnetree/commit/47a4729e2e2361f9cc8530e533cf1e83e76c1027))

## [0.6.3](https://github.com/HerbHall/subnetree/compare/v0.6.2...v0.6.3) (2026-04-07)


### Features

* allow renaming device hostname from detail page ([#555](https://github.com/HerbHall/subnetree/issues/555)) ([32e1a39](https://github.com/HerbHall/subnetree/commit/32e1a396ba08f0e45aaeda633c773bb3979f6a40))
* collapse unclassified devices in topology view ([#563](https://github.com/HerbHall/subnetree/issues/563)) ([e24db1a](https://github.com/HerbHall/subnetree/commit/e24db1a5e8c066a42d751d3a6c0806b269b32cac))
* infer gateway topology links + unified toolbar + UI improvements ([#562](https://github.com/HerbHall/subnetree/issues/562)) ([16494c5](https://github.com/HerbHall/subnetree/commit/16494c56ab08dcdcc7a5d61f06c831e4f3735d99))


### Bug Fixes

* align status.md phase with Samverk lifecycle naming ([e28988f](https://github.com/HerbHall/subnetree/commit/e28988f6e2d23a9f49fe0a3f913e059e07f309a6))
* pin Dockerfile Go image to 1.25.8 to match go.mod ([#557](https://github.com/HerbHall/subnetree/issues/557)) ([8c0657b](https://github.com/HerbHall/subnetree/commit/8c0657ba416f3f4f45280c6957e69b261065b570))
* repair corrupted CODEOWNERS and verify Copilot auto-review setup ([#556](https://github.com/HerbHall/subnetree/issues/556)) ([c121ee2](https://github.com/HerbHall/subnetree/commit/c121ee23ebd0a649d4d13832d0251b0fa2e2c574))
* resolve govulncheck CI failures blocking all PRs ([#552](https://github.com/HerbHall/subnetree/issues/552)) ([bf4a965](https://github.com/HerbHall/subnetree/commit/bf4a96565d626237e2e9b11272f2359638ba9bea))

## [0.6.2](https://github.com/HerbHall/subnetree/compare/v0.6.1...v0.6.2) (2026-03-07)


### Features

* active WiFi network scanning via OS APIs ([#461](https://github.com/HerbHall/subnetree/issues/461)) ([7cdc127](https://github.com/HerbHall/subnetree/commit/7cdc1279029dfba8aa6145bb2fbba6c66bfcd6de))
* add contextual help and tooltips across UI ([#454](https://github.com/HerbHall/subnetree/issues/454)) ([770ea9e](https://github.com/HerbHall/subnetree/commit/770ea9ee1a614cf76c4e5d72e4c3bacf08ecbd71))
* add contextual help to devices page ([#455](https://github.com/HerbHall/subnetree/issues/455)) ([698be78](https://github.com/HerbHall/subnetree/commit/698be783f7bc0998a59d82be5e12dc362be9e7e5))
* add Cosign keyless container image signing ([#481](https://github.com/HerbHall/subnetree/issues/481)) ([161956e](https://github.com/HerbHall/subnetree/commit/161956ea725107c8d8fe05caf8748a769ca97686))
* add custom background support to topology page ([#484](https://github.com/HerbHall/subnetree/issues/484)) ([e51911d](https://github.com/HerbHall/subnetree/commit/e51911d5db2e20a082ac67818686a22ef612a1e1))
* add get_stale_devices and get_service_inventory MCP tools ([#469](https://github.com/HerbHall/subnetree/issues/469)) ([b6e8bfd](https://github.com/HerbHall/subnetree/commit/b6e8bfd3ccd6c42dd6b9298406144996e6230650))
* add GitHub Copilot integration ([#513](https://github.com/HerbHall/subnetree/issues/513)) ([0829288](https://github.com/HerbHall/subnetree/commit/0829288d579129f2db9c7315e61539e782144352))
* add Holt-Winters seasonal baselines to Insight module ([#482](https://github.com/HerbHall/subnetree/issues/482)) ([30f34c9](https://github.com/HerbHall/subnetree/commit/30f34c91c87b6c58825ad2c7907dd95d5b6c2032))
* add Home Assistant MQTT auto-discovery ([#501](https://github.com/HerbHall/subnetree/issues/501)) ([b57efa7](https://github.com/HerbHall/subnetree/commit/b57efa76495d77cce60933639ef59cbcd8edfedc))
* add location, role, and last seen columns to devices table ([#470](https://github.com/HerbHall/subnetree/issues/470)) ([0ce10c6](https://github.com/HerbHall/subnetree/commit/0ce10c626a95899b29b9d96d8035ab4f983c60d6))
* add MCP audit log for tool call tracking ([#472](https://github.com/HerbHall/subnetree/issues/472)) ([01e0117](https://github.com/HerbHall/subnetree/commit/01e0117199a00af2dd87ff096719c8ab57929316))
* add NetBox DCIM export module ([#500](https://github.com/HerbHall/subnetree/issues/500)) ([23a49e0](https://github.com/HerbHall/subnetree/commit/23a49e0cbb0a4e6883bdded09a91b9f3cf50d2e5))
* add per-device Markdown documentation to autodoc engine ([#491](https://github.com/HerbHall/subnetree/issues/491)) ([97a2e3a](https://github.com/HerbHall/subnetree/commit/97a2e3aa5e0f6454a4e0197faf2edd8b86a5e656))
* add release gate and nightly build workflows ([#525](https://github.com/HerbHall/subnetree/issues/525)) ([963f08d](https://github.com/HerbHall/subnetree/commit/963f08d995ce1887cb01a7f7803a1ff8d7760534))
* add Scout agent auto-update mechanism ([#474](https://github.com/HerbHall/subnetree/issues/474)) ([fd3452b](https://github.com/HerbHall/subnetree/commit/fd3452b406fa1c684143320bc1affaf456cd86a6))
* add topology-aware alert correlation engine ([#483](https://github.com/HerbHall/subnetree/issues/483)) ([f64ef28](https://github.com/HerbHall/subnetree/commit/f64ef28dea6ab4cf3f741e583684e3eaf83a3ea4))
* add Trivy container vulnerability scanning to CI and release ([#480](https://github.com/HerbHall/subnetree/issues/480)) ([d673db3](https://github.com/HerbHall/subnetree/commit/d673db3ecf2319ccb4a30c31d49c2697dfee709b))
* **auth:** add MFA/TOTP two-factor authentication ([#466](https://github.com/HerbHall/subnetree/issues/466)) ([fe06f1a](https://github.com/HerbHall/subnetree/commit/fe06f1a2d0e8c006786ff758f398d8f0ffe6abf4))
* CI smoke test of Docker image after GoReleaser ([#400](https://github.com/HerbHall/subnetree/issues/400)) ([4b4ee13](https://github.com/HerbHall/subnetree/commit/4b4ee132c948c4d9125ad2008bbdb9e34b97d677))
* enhance Scout installer with service registration ([#453](https://github.com/HerbHall/subnetree/issues/453)) ([65cdf7a](https://github.com/HerbHall/subnetree/commit/65cdf7a13b5b96824ddd26fae00411e5fe698fe2))
* group devices by subnet with collapsible headers ([#452](https://github.com/HerbHall/subnetree/issues/452)) ([120ed3b](https://github.com/HerbHall/subnetree/commit/120ed3be1f3875cb6a075158ee67727c940aa033))
* hardware asset profile API and event bridge ([#442](https://github.com/HerbHall/subnetree/issues/442)) ([d7fad5a](https://github.com/HerbHall/subnetree/commit/d7fad5a89719472ffea1f235afe7a4ce5c1dca67))
* hardware asset profile schema and store layer ([#437](https://github.com/HerbHall/subnetree/issues/437)) ([#441](https://github.com/HerbHall/subnetree/issues/441)) ([48db549](https://github.com/HerbHall/subnetree/commit/48db549711023c19c23d03d01b2ab3194bbd6d3e))
* hardware profile UI and seed data ([#437](https://github.com/HerbHall/subnetree/issues/437)) ([#443](https://github.com/HerbHall/subnetree/issues/443)) ([265af9a](https://github.com/HerbHall/subnetree/commit/265af9a0551866a81fee6379a5019dcdae43a4ec))
* ICMP traceroute implementation with API endpoint ([#402](https://github.com/HerbHall/subnetree/issues/402)) ([29dd19a](https://github.com/HerbHall/subnetree/commit/29dd19af1f145b1a516fee20a1c8400bc0fd7054))
* Inno Setup installer for Scout agent on Windows ([#444](https://github.com/HerbHall/subnetree/issues/444)) ([457ab8f](https://github.com/HerbHall/subnetree/commit/457ab8fffdbc34812e5a3abe256be4854b01cded))
* interactive diagnostic tools for device troubleshooting ([#405](https://github.com/HerbHall/subnetree/issues/405)) ([15c7c9c](https://github.com/HerbHall/subnetree/commit/15c7c9c9390493c0910eec31f790cd29a47bec2a))
* MCP server interface for AI tool integration ([#445](https://github.com/HerbHall/subnetree/issues/445)) ([c137db0](https://github.com/HerbHall/subnetree/commit/c137db003df8989d04041b6befa40efb2ec7acd4))
* network hierarchy inference from scan data ([#408](https://github.com/HerbHall/subnetree/issues/408)) ([617ec66](https://github.com/HerbHall/subnetree/commit/617ec66e0d63c8df6e7447d6eae73d09194fdc8e))
* one-click Scout agent deployment ([#432](https://github.com/HerbHall/subnetree/issues/432)) ([3a9c4c2](https://github.com/HerbHall/subnetree/commit/3a9c4c28282ee4fe938f8037fcb8151f23c240cf))
* Playwright E2E tests for critical user paths ([#409](https://github.com/HerbHall/subnetree/issues/409)) ([92ab6c8](https://github.com/HerbHall/subnetree/commit/92ab6c8409657bcba159c29479dd5b07da69977d))
* Proxmox VE VM/LXC inventory and resource monitoring ([#463](https://github.com/HerbHall/subnetree/issues/463)) ([d007061](https://github.com/HerbHall/subnetree/commit/d0070610616894e16234266adb6cbe11b3a215ce))
* public demo mode with read-only access and seed data ([#462](https://github.com/HerbHall/subnetree/issues/462)) ([22c19dc](https://github.com/HerbHall/subnetree/commit/22c19dc3291457483b035404e4e6f573d676ddbd))
* Scout GPU collection, Proxmox VE collector, hardware refresh ([#446](https://github.com/HerbHall/subnetree/issues/446)) ([5aa1705](https://github.com/HerbHall/subnetree/commit/5aa1705dff8aef8039c4d6ab1670a9a1acd259bc))
* seed data for staging and demo environments ([#404](https://github.com/HerbHall/subnetree/issues/404)) ([baf620f](https://github.com/HerbHall/subnetree/commit/baf620f904f1e8c2a2cc073b71c388604a54f249))
* SNMP FDB table walks for switch port mapping ([#403](https://github.com/HerbHall/subnetree/issues/403)) ([56e3655](https://github.com/HerbHall/subnetree/commit/56e3655c1eba902599304267a90c395622cb942c))
* store classification confidence on device model ([#401](https://github.com/HerbHall/subnetree/issues/401)) ([d53c043](https://github.com/HerbHall/subnetree/commit/d53c043eaa636278f4198bb61758fb6207003f26))
* Tailscale overlay network plugin ([#465](https://github.com/HerbHall/subnetree/issues/465)) ([e4089d8](https://github.com/HerbHall/subnetree/commit/e4089d88d2f604786c7e45e75fa67fcfad2612b0))
* WiFi heuristic device tagging ([#355](https://github.com/HerbHall/subnetree/issues/355) Phase A) ([#458](https://github.com/HerbHall/subnetree/issues/458)) ([400bca2](https://github.com/HerbHall/subnetree/commit/400bca244867be1e31718ae383bd12adff12172e))
* WiFi hotspot client enumeration for AP-mode servers ([#464](https://github.com/HerbHall/subnetree/issues/464)) ([91fefe6](https://github.com/HerbHall/subnetree/commit/91fefe65d024ad2398b561b7ed01eab2b3a0c0a4))
* Windows Scout service mode with install/uninstall subcommands ([#451](https://github.com/HerbHall/subnetree/issues/451)) ([464690d](https://github.com/HerbHall/subnetree/commit/464690d427c03d291ce7d28e08fec40ac42f02bd))


### Bug Fixes

* add light theme overrides for topology and chart colors ([#420](https://github.com/HerbHall/subnetree/issues/420)) ([0981811](https://github.com/HerbHall/subnetree/commit/0981811fbfce346e9b2b96c51d0edc73c7869388))
* add missing playwright peer dependency ([#504](https://github.com/HerbHall/subnetree/issues/504)) ([bc4e586](https://github.com/HerbHall/subnetree/commit/bc4e586ceb4489c0f83c0a124fb677cda8ff4eeb))
* agent pages - setup link in empty state, shell labels on code blocks ([#425](https://github.com/HerbHall/subnetree/issues/425)) ([3197cf5](https://github.com/HerbHall/subnetree/commit/3197cf509bedc5d3fc81f830351098758680fcdc))
* compact device table rows, default sort by IP, fix topology panel overlap ([#422](https://github.com/HerbHall/subnetree/issues/422)) ([568e568](https://github.com/HerbHall/subnetree/commit/568e5683ae3cfca183660fdd3e8ba99ed72c1f35))
* disable mDNS/UPnP auto-discovery in QC container ([#419](https://github.com/HerbHall/subnetree/issues/419)) ([76c5aca](https://github.com/HerbHall/subnetree/commit/76c5aca8d38c504f31a3782733a375cc1bda2980))
* increase default page size to 256 for full Class C ([#424](https://github.com/HerbHall/subnetree/issues/424)) ([867cc5c](https://github.com/HerbHall/subnetree/commit/867cc5cfeb7d30a48ac6dcca2a20120198100545))
* merge all metadata fields in UpsertDevice update path ([#417](https://github.com/HerbHall/subnetree/issues/417)) ([3a6145c](https://github.com/HerbHall/subnetree/commit/3a6145c3152264ca1c05ba00a02a19153cc9bde3))
* replace deprecated wmic with PowerShell CIM for profiling ([#456](https://github.com/HerbHall/subnetree/issues/456)) ([dfb7395](https://github.com/HerbHall/subnetree/commit/dfb7395aaf6577a1f4284958d82858e95e307efc))
* use global inventory counts for status filter pills ([#418](https://github.com/HerbHall/subnetree/issues/418)) ([839a091](https://github.com/HerbHall/subnetree/commit/839a091e3bf4f80b0331b46be871235fa658049a))
