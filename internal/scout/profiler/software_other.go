//go:build !windows

package profiler

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"go.uber.org/zap"
)

const maxPackages = 500

func collectSoftware(ctx context.Context, logger *zap.Logger) (*scoutpb.SoftwareInventory, error) {
	sw := &scoutpb.SoftwareInventory{
		OsName: runtime.GOOS,
	}

	// OS details from /etc/os-release.
	osName, osVersion, osBuild := readOSRelease(logger)
	if osName != "" {
		sw.OsName = osName
	}
	sw.OsVersion = osVersion
	sw.OsBuild = osBuild

	// Installed packages via dpkg or rpm.
	sw.Packages = collectLinuxPackages(ctx, logger)

	// Docker containers (best-effort).
	sw.DockerContainers = collectDockerContainers(ctx, logger)

	return sw, nil
}

// readOSRelease parses /etc/os-release for OS identification.
func readOSRelease(logger *zap.Logger) (name, version, build string) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		logger.Debug("failed to read /etc/os-release", zap.Error(err))
		return runtime.GOOS, runtime.GOARCH, ""
	}

	fields := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		val := strings.Trim(parts[1], `"`)
		fields[key] = val
	}

	// Prefer PRETTY_NAME, fall back to NAME.
	name = fields["PRETTY_NAME"]
	if name == "" {
		name = fields["NAME"]
	}
	if name == "" {
		name = runtime.GOOS
	}

	version = fields["VERSION_ID"]
	build = fields["BUILD_ID"]

	return name, version, build
}

// collectLinuxPackages tries dpkg (Debian/Ubuntu) then rpm (RHEL/Fedora).
func collectLinuxPackages(ctx context.Context, logger *zap.Logger) []*scoutpb.InstalledPackage {
	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Try dpkg first (Debian/Ubuntu).
	packages := tryDpkg(cmdCtx, logger)
	if packages != nil {
		return packages
	}

	// Try rpm (RHEL/Fedora/SUSE).
	return tryRPM(cmdCtx, logger)
}

func tryDpkg(ctx context.Context, logger *zap.Logger) []*scoutpb.InstalledPackage {
	cmd := exec.CommandContext(ctx, "dpkg-query", "-W",
		"-f", "${Package}\t${Version}\t${Maintainer}\t${db:Status-Abbrev}\n")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		logger.Debug("dpkg-query not available", zap.Error(err))
		return nil
	}

	var packages []*scoutpb.InstalledPackage
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		if len(packages) >= maxPackages {
			break
		}
		line := scanner.Text()
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 2 {
			continue
		}

		// Only include installed packages (status starts with "ii").
		if len(parts) >= 4 && !strings.HasPrefix(parts[3], "ii") {
			continue
		}

		pkg := &scoutpb.InstalledPackage{
			Name:    parts[0],
			Version: parts[1],
		}
		if len(parts) >= 3 {
			pkg.Publisher = parts[2]
		}
		packages = append(packages, pkg)
	}

	logger.Debug("collected dpkg packages", zap.Int("count", len(packages)))
	return packages
}

func tryRPM(ctx context.Context, logger *zap.Logger) []*scoutpb.InstalledPackage {
	cmd := exec.CommandContext(ctx, "rpm", "-qa",
		"--queryformat", "%{NAME}\t%{VERSION}-%{RELEASE}\t%{VENDOR}\t\n")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		logger.Debug("rpm not available", zap.Error(err))
		return nil
	}

	var packages []*scoutpb.InstalledPackage
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		if len(packages) >= maxPackages {
			break
		}
		line := scanner.Text()
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 2 {
			continue
		}

		pkg := &scoutpb.InstalledPackage{
			Name:    parts[0],
			Version: parts[1],
		}
		if len(parts) >= 3 {
			pkg.Publisher = parts[2]
		}
		packages = append(packages, pkg)
	}

	logger.Debug("collected rpm packages", zap.Int("count", len(packages)))
	return packages
}

// collectDockerContainers runs docker ps to list running containers.
func collectDockerContainers(ctx context.Context, logger *zap.Logger) []*scoutpb.DockerContainer {
	cmdCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "docker", "ps", "--format", "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		logger.Debug("docker ps failed (docker may not be installed)", zap.Error(err))
		return nil
	}

	lines := strings.Split(stdout.String(), "\n")
	containers := make([]*scoutpb.DockerContainer, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		containers = append(containers, &scoutpb.DockerContainer{
			ContainerId: parts[0],
			Name:        parts[1],
			Image:       parts[2],
			Status:      parts[3],
		})
	}

	logger.Debug("collected docker containers", zap.Int("count", len(containers)))
	return containers
}
