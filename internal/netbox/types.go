package netbox

// NetBox API request/response types.
// These mirror the NetBox v4 REST API entity shapes used by the sync client.

// NBSite represents a NetBox site (data center / location).
type NBSite struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Status string `json:"status,omitempty"`
	URL    string `json:"url,omitempty"`
}

// NBManufacturer represents a NetBox manufacturer.
type NBManufacturer struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	URL  string `json:"url,omitempty"`
}

// NBDeviceType represents a NetBox device type (hardware model).
type NBDeviceType struct {
	ID           int             `json:"id"`
	Manufacturer *NBManufacturer `json:"manufacturer,omitempty"`
	Model        string          `json:"model"`
	Slug         string          `json:"slug"`
	URL          string          `json:"url,omitempty"`
}

// NBDeviceRole represents a NetBox device role (functional purpose).
type NBDeviceRole struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Color string `json:"color,omitempty"`
	URL   string `json:"url,omitempty"`
}

// NBTag represents a NetBox tag.
type NBTag struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Color string `json:"color,omitempty"`
	URL   string `json:"url,omitempty"`
}

// NBDevice represents a NetBox device entity.
type NBDevice struct {
	ID           int                    `json:"id"`
	Name         string                 `json:"name"`
	DeviceType   *NBDeviceType          `json:"device_type,omitempty"`
	Role         *NBDeviceRole          `json:"role,omitempty"`
	Site         *NBSite                `json:"site,omitempty"`
	Status       *NBStatusValue         `json:"status,omitempty"`
	Serial       string                 `json:"serial,omitempty"`
	Comments     string                 `json:"comments,omitempty"`
	Tags         []NBTag                `json:"tags,omitempty"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
	URL          string                 `json:"url,omitempty"`
}

// NBStatusValue represents a NetBox status choice (value + label).
type NBStatusValue struct {
	Value string `json:"value"`
	Label string `json:"label,omitempty"`
}

// NBInterface represents a NetBox device interface.
type NBInterface struct {
	ID         int        `json:"id"`
	Device     *NBDevice  `json:"device,omitempty"`
	Name       string     `json:"name"`
	Type       *NBTypeVal `json:"type,omitempty"`
	MACAddress string     `json:"mac_address,omitempty"`
	URL        string     `json:"url,omitempty"`
}

// NBTypeVal represents a NetBox type choice (value + label).
type NBTypeVal struct {
	Value string `json:"value"`
	Label string `json:"label,omitempty"`
}

// NBIPAddress represents a NetBox IP address assignment.
type NBIPAddress struct {
	ID                 int    `json:"id"`
	Address            string `json:"address"`
	AssignedObjectType string `json:"assigned_object_type,omitempty"`
	AssignedObjectID   int    `json:"assigned_object_id,omitempty"`
	URL                string `json:"url,omitempty"`
}

// ListResponse is the generic paginated response from NetBox list endpoints.
type ListResponse[T any] struct {
	Count    int    `json:"count"`
	Next     string `json:"next,omitempty"`
	Previous string `json:"previous,omitempty"`
	Results  []T    `json:"results"`
}

// SyncResult summarizes the outcome of a sync operation.
type SyncResult struct {
	Created int      `json:"created"`
	Updated int      `json:"updated"`
	Skipped int      `json:"skipped"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
	DryRun  bool     `json:"dry_run"`
}

// NBDeviceCreateRequest is the payload for creating/updating a NetBox device.
type NBDeviceCreateRequest struct {
	Name       string                 `json:"name"`
	DeviceType int                    `json:"device_type"`
	Role       int                    `json:"role"`
	Site       int                    `json:"site"`
	Status     string                 `json:"status"`
	Serial     string                 `json:"serial,omitempty"`
	Comments   string                 `json:"comments,omitempty"`
	Tags       []int                  `json:"tags,omitempty"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
}

// NBInterfaceCreateRequest is the payload for creating a NetBox interface.
type NBInterfaceCreateRequest struct {
	Device     int    `json:"device"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	MACAddress string `json:"mac_address,omitempty"`
}

// NBIPAddressCreateRequest is the payload for creating a NetBox IP address.
type NBIPAddressCreateRequest struct {
	Address            string `json:"address"`
	AssignedObjectType string `json:"assigned_object_type,omitempty"`
	AssignedObjectID   int    `json:"assigned_object_id,omitempty"`
}
