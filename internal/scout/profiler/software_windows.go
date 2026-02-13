//go:build windows

package profiler

import (
	"bytes"
	"context"
	"os/exec"
	"runtime"
	"strings"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"go.uber.org/zap"
	"golang.org/x/sys/windows/registry"
)

func collectSoftware(ctx context.Context, logger *zap.Logger) (*scoutpb.SoftwareInventory, error) {
	sw := &scoutpb.SoftwareInventory{
		OsName: runtime.GOOS,
	}

	// OS details via wmic.
	osRows, err := runWMIC(ctx, "os", "Caption,Version,BuildNumber")
	if err != nil {
		logger.Debug("wmic os failed", zap.Error(err))
	} else if len(osRows) > 0 {
		row := osRows[0]
		if caption := row["Caption"]; caption != "" {
			sw.OsName = caption
		}
		sw.OsVersion = row["Version"]
		sw.OsBuild = row["BuildNumber"]
	}

	// Installed packages from Windows registry.
	packages := readInstalledPackages(logger)
	sw.Packages = packages

	// Docker containers (best-effort).
	containers := collectDockerContainers(ctx, logger)
	sw.DockerContainers = containers

	return sw, nil
}

// readInstalledPackages reads installed programs from the Windows registry.
func readInstalledPackages(logger *zap.Logger) []*scoutpb.InstalledPackage {
	var packages []*scoutpb.InstalledPackage

	paths := []struct {
		root registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`},
	}

	seen := make(map[string]bool)

	for _, p := range paths {
		key, err := registry.OpenKey(p.root, p.path, registry.READ)
		if err != nil {
			continue
		}

		subkeys, err := key.ReadSubKeyNames(-1)
		key.Close()
		if err != nil {
			continue
		}

		for _, sk := range subkeys {
			subKey, err := registry.OpenKey(p.root, p.path+`\`+sk, registry.READ)
			if err != nil {
				continue
			}

			name, _, _ := subKey.GetStringValue("DisplayName")
			version, _, _ := subKey.GetStringValue("DisplayVersion")
			publisher, _, _ := subKey.GetStringValue("Publisher")
			installDate, _, _ := subKey.GetStringValue("InstallDate")
			subKey.Close()

			if name == "" {
				continue
			}

			// Deduplicate by name+version.
			dedup := name + "|" + version
			if seen[dedup] {
				continue
			}
			seen[dedup] = true

			packages = append(packages, &scoutpb.InstalledPackage{
				Name:        name,
				Version:     version,
				Publisher:   publisher,
				InstallDate: installDate,
			})
		}
	}

	logger.Debug("collected installed packages", zap.Int("count", len(packages)))
	return packages
}

// collectDockerContainers runs docker ps to list running containers.
func collectDockerContainers(ctx context.Context, logger *zap.Logger) []*scoutpb.DockerContainer {
	cmd := exec.CommandContext(ctx, "docker", "ps", "--format", "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		// Docker may not be installed; that is fine.
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
