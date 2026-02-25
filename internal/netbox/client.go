package netbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client wraps the NetBox REST API v4.
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// NewClient creates a new NetBox API client.
func NewClient(baseURL, token string, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
	}
}

// ListDevicesByTag retrieves all devices that have the specified tag.
func (c *Client) ListDevicesByTag(ctx context.Context, tag string) ([]NBDevice, error) {
	path := fmt.Sprintf("/api/dcim/devices/?tag=%s&limit=1000", tag)
	var resp ListResponse[NBDevice]
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("list devices by tag: %w", err)
	}
	return resp.Results, nil
}

// CreateDevice creates a new device in NetBox.
func (c *Client) CreateDevice(ctx context.Context, req NBDeviceCreateRequest) (*NBDevice, error) {
	var device NBDevice
	if err := c.doJSON(ctx, http.MethodPost, "/api/dcim/devices/", req, &device); err != nil {
		return nil, fmt.Errorf("create device: %w", err)
	}
	return &device, nil
}

// UpdateDevice patches an existing device in NetBox.
func (c *Client) UpdateDevice(ctx context.Context, id int, req NBDeviceCreateRequest) (*NBDevice, error) {
	path := fmt.Sprintf("/api/dcim/devices/%d/", id)
	var device NBDevice
	if err := c.doJSON(ctx, http.MethodPatch, path, req, &device); err != nil {
		return nil, fmt.Errorf("update device %d: %w", id, err)
	}
	return &device, nil
}

// GetOrCreateManufacturer finds a manufacturer by name or creates it.
func (c *Client) GetOrCreateManufacturer(ctx context.Context, name string) (int, error) {
	slug := SlugFromName(name)
	path := fmt.Sprintf("/api/dcim/manufacturers/?slug=%s", slug)
	var resp ListResponse[NBManufacturer]
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return 0, fmt.Errorf("list manufacturers: %w", err)
	}
	if resp.Count > 0 {
		return resp.Results[0].ID, nil
	}

	body := map[string]string{"name": name, "slug": slug}
	var created NBManufacturer
	if err := c.doJSON(ctx, http.MethodPost, "/api/dcim/manufacturers/", body, &created); err != nil {
		return 0, fmt.Errorf("create manufacturer %q: %w", name, err)
	}
	return created.ID, nil
}

// GetOrCreateDeviceType finds a device type by manufacturer and model or creates it.
func (c *Client) GetOrCreateDeviceType(ctx context.Context, manufacturerID int, model string) (int, error) {
	slug := SlugFromName(model)
	path := fmt.Sprintf("/api/dcim/device-types/?manufacturer_id=%d&slug=%s", manufacturerID, slug)
	var resp ListResponse[NBDeviceType]
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return 0, fmt.Errorf("list device types: %w", err)
	}
	if resp.Count > 0 {
		return resp.Results[0].ID, nil
	}

	body := map[string]interface{}{
		"manufacturer": manufacturerID,
		"model":        model,
		"slug":         slug,
	}
	var created NBDeviceType
	if err := c.doJSON(ctx, http.MethodPost, "/api/dcim/device-types/", body, &created); err != nil {
		return 0, fmt.Errorf("create device type %q: %w", model, err)
	}
	return created.ID, nil
}

// GetOrCreateDeviceRole finds a device role by name or creates it.
func (c *Client) GetOrCreateDeviceRole(ctx context.Context, name string) (int, error) {
	slug := SlugFromName(name)
	path := fmt.Sprintf("/api/dcim/device-roles/?slug=%s", slug)
	var resp ListResponse[NBDeviceRole]
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return 0, fmt.Errorf("list device roles: %w", err)
	}
	if resp.Count > 0 {
		return resp.Results[0].ID, nil
	}

	body := map[string]string{"name": name, "slug": slug, "color": "9e9e9e"}
	var created NBDeviceRole
	if err := c.doJSON(ctx, http.MethodPost, "/api/dcim/device-roles/", body, &created); err != nil {
		return 0, fmt.Errorf("create device role %q: %w", name, err)
	}
	return created.ID, nil
}

// GetOrCreateSite finds a site by name or creates it.
func (c *Client) GetOrCreateSite(ctx context.Context, name string) (int, error) {
	slug := SlugFromName(name)
	path := fmt.Sprintf("/api/dcim/sites/?slug=%s", slug)
	var resp ListResponse[NBSite]
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return 0, fmt.Errorf("list sites: %w", err)
	}
	if resp.Count > 0 {
		return resp.Results[0].ID, nil
	}

	body := map[string]string{"name": name, "slug": slug, "status": "active"}
	var created NBSite
	if err := c.doJSON(ctx, http.MethodPost, "/api/dcim/sites/", body, &created); err != nil {
		return 0, fmt.Errorf("create site %q: %w", name, err)
	}
	return created.ID, nil
}

// CreateInterface creates a network interface on a device.
func (c *Client) CreateInterface(ctx context.Context, deviceID int, name, macAddr string) (*NBInterface, error) {
	req := NBInterfaceCreateRequest{
		Device:     deviceID,
		Name:       name,
		Type:       "other",
		MACAddress: macAddr,
	}
	var iface NBInterface
	if err := c.doJSON(ctx, http.MethodPost, "/api/dcim/interfaces/", req, &iface); err != nil {
		return nil, fmt.Errorf("create interface: %w", err)
	}
	return &iface, nil
}

// CreateIPAddress creates an IP address and optionally assigns it to an interface.
func (c *Client) CreateIPAddress(ctx context.Context, address string, interfaceID int) (*NBIPAddress, error) {
	req := NBIPAddressCreateRequest{
		Address: address,
	}
	if interfaceID > 0 {
		req.AssignedObjectType = "dcim.interface"
		req.AssignedObjectID = interfaceID
	}
	var ip NBIPAddress
	if err := c.doJSON(ctx, http.MethodPost, "/api/ipam/ip-addresses/", req, &ip); err != nil {
		return nil, fmt.Errorf("create ip address: %w", err)
	}
	return &ip, nil
}

// EnsureTag finds a tag by name or creates it. Returns the tag ID.
func (c *Client) EnsureTag(ctx context.Context, name string) (int, error) {
	slug := SlugFromName(name)
	path := fmt.Sprintf("/api/extras/tags/?slug=%s", slug)
	var resp ListResponse[NBTag]
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return 0, fmt.Errorf("list tags: %w", err)
	}
	if resp.Count > 0 {
		return resp.Results[0].ID, nil
	}

	body := map[string]string{"name": name, "slug": slug, "color": "4caf50"}
	var created NBTag
	if err := c.doJSON(ctx, http.MethodPost, "/api/extras/tags/", body, &created); err != nil {
		return 0, fmt.Errorf("create tag %q: %w", name, err)
	}
	return created.ID, nil
}

// doJSON performs an HTTP request with JSON serialization/deserialization.
func (c *Client) doJSON(ctx context.Context, method, path string, body, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Token "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("netbox API %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}
