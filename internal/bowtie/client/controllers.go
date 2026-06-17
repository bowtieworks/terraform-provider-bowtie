package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// TaggedValue models a Bowtie internally-tagged enum that serializes as
// {"type": "<variant>", "value": <inner>}. Variants without an inner value
// (for example version_strategy "manual" or wireguard_strategy "static") omit
// the value member entirely.
type TaggedValue struct {
	Type  string  `json:"type"`
	Value *string `json:"value,omitempty"`
}

// ControllerSettings is the full ControllerRepresentation as served by
// GET/POST /organization/controller. The update endpoint overlays the posted
// payload onto the existing record, so callers must read the current
// representation, change only the fields they manage, and post the whole thing
// back. Server-computed and unmanaged fields are kept as json.RawMessage so
// they round-trip unchanged.
type ControllerSettings struct {
	ID                              string          `json:"id"`
	SiteID                          *string         `json:"site_id"`
	PublicAddress                   string          `json:"public_address"`
	SyncState                       json.RawMessage `json:"sync_state,omitempty"`
	Status                          json.RawMessage `json:"status,omitempty"`
	Features                        []string        `json:"features,omitempty"`
	WireguardPort                   int             `json:"wireguard_port"`
	WireguardAddress                *string         `json:"wireguard_address"`
	PublicKey                       string          `json:"public_key"`
	HTTPSEndpoint                   string          `json:"https_endpoint"`
	PersistentKeepalive             int             `json:"persistent_keepalive"`
	DeviceID                        *string         `json:"device_id"`
	IPV6                            *string         `json:"ipv6"`
	IPV4                            json.RawMessage `json:"ipv4,omitempty"`
	SyncAddress                     *string         `json:"sync_address"`
	VersionStrategy                 TaggedValue     `json:"version_strategy"`
	VersionStrategySplay            *TaggedValue    `json:"version_strategy_splay"`
	VersionIncludePrereleases       *bool           `json:"version_include_prereleases"`
	VersionMinimumAge               *int            `json:"version_minimum_age"`
	WireguardStrategy               TaggedValue     `json:"wireguard_strategy"`
	BackupStrategies                json.RawMessage `json:"backup_strategies,omitempty"`
	CanUseVanityDomain              bool            `json:"can_use_vanity_domain"`
	CanUsePublicHTTPS               bool            `json:"can_use_public_https"`
	CanUseIDP                       bool            `json:"can_use_idp"`
	CurrentVersion                  *string         `json:"current_version"`
	TrackPolicyVerdictMetrics       *bool           `json:"track_policy_verdict_metrics"`
	TrackPolicyVerdictLogs          *bool           `json:"track_policy_verdict_logs"`
	WebFilterTrustedProxyCollection *string         `json:"web_filter_trusted_proxy_collection"`
	MinimumPeersBehavior            json.RawMessage `json:"minimum_peers_behavior,omitempty"`
	AllowTemporaryConsoleUsers      bool            `json:"allow_temporary_console_users"`
	SSHListener                     *string         `json:"ssh_listener"`
}

// ListControllers returns every Controller registered in the organization.
func (c *Client) ListControllers() ([]ControllerSettings, error) {
	var controllers []ControllerSettings
	if err := c.getListJSON(&controllers, "/organization/controller", "/organization/controller/"); err != nil {
		return nil, err
	}
	return controllers, nil
}

// GetController reads a single Controller's full representation by ID.
func (c *Client) GetController(id string) (*ControllerSettings, error) {
	var controller ControllerSettings
	if err := c.getListJSON(&controller, fmt.Sprintf("/organization/controller/%s", id)); err != nil {
		return nil, err
	}
	return &controller, nil
}

// UpdateController posts a full ControllerRepresentation. The Controller must
// already exist (Controllers self-register at boot); the endpoint returns 404
// otherwise. The id member of the payload selects the Controller to update.
func (c *Client) UpdateController(settings *ControllerSettings) (*ControllerSettings, error) {
	body, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.getHostURL("/organization/controller"), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	respBody, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var updated ControllerSettings
	if err := json.Unmarshal(respBody, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

// DeleteController removes a Controller record from the control plane.
func (c *Client) DeleteController(id string) error {
	req, err := http.NewRequest(http.MethodDelete, c.getHostURL(fmt.Sprintf("/organization/controller/%s", id)), nil)
	if err != nil {
		return err
	}
	_, err = c.doRequest(req)
	return err
}
