package client

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Organization struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Domain     string         `json:"domain"`
	DNS        map[string]DNS `json:"dns"`
	IPV6Ranges []string       `json:"ipv6_ranges"`
	Sites      []Site         `json:"sites"`
}

type OrganizationPayload struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

type Site struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	RoutableRangesV4 []RoutableRange `json:"routeable_ranges_v4,omitempty"`
	RouteRangesV6    []RoutableRange `json:"routeable_ranges_v6,omitempty"`
	Controllers      []Controller    `json:"controllers,omitempty"`
}

type RoutableRange struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Range       string `json:"range"`
	Weight      int64  `json:"weight"`
	Metric      int64  `json:"metric"`
	Description string `json:"description"`
	ISV4        bool   `json:"is_v4,omitempty"`
	ISV6        bool   `json:"is_v6,omitempty"`
}

type DNS struct {
	ID               string                `json:"id"`
	Name             string                `json:"name"`
	IsDNS64          bool                  `json:"is_dns64"`
	Servers          map[string]Server     `json:"servers"`
	IncludeOnlySites []string              `json:"include_only_sites"`
	IsCounted        bool                  `json:"is_counted"`
	IsLog            bool                  `json:"is_log"`
	IsDropA          bool                  `json:"is_drop_a"`
	IsDropAll        bool                  `json:"is_drop_all"`
	IsSearchDomain   bool                  `json:"is_search_domain"`
	DNS64Exclude     map[string]DNSExclude `json:"dns64_exclude"`
}

type DNSBlockList struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Upstream        string `json:"upstream,omitempty"`
	OverrideToAllow string `json:"override_to_allow"`
	IsAllowlist     bool   `json:"is_allowlist"`
}

type Server struct {
	ID    string `json:"id"`
	Addr  string `json:"addr"`
	Order int64  `json:"order"`
}

type DNSExclude struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Order int64  `json:"order"`
}

type Controller struct {
	ID                  string   `json:"id"`
	SiteID              string   `json:"site_id"`
	PublicAddress       string   `json:"public_address"`
	SyncAddress         string   `json:"sync_address"`
	SyncState           string   `json:"sync_state"`
	Status              string   `json:"status"`
	Features            []string `json:"features"`
	WireguardPort       int      `json:"wireguard_port"`
	PublicKey           string   `json:"public_key"`
	HTTPSEndpoint       string   `json:"https_endpoint"`
	PersistentKeepalive int      `json:"persistent_keepalive"`
	DeviceID            string   `json:"device_id"`
	IPV6                string   `json:"ipv6"`
}

func (c *Client) GetOrganization() (*Organization, error) {
	req, err := http.NewRequest(http.MethodGet, c.getHostURL("/organization"), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var org *Organization = &Organization{}
	err = json.Unmarshal(body, org)

	return org, err
}

func (c *Client) UpsertOrganization(name string, domain string) error {
	payload := OrganizationPayload{
		Name:   name,
		Domain: domain,
	}

	requestPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.getHostURL("/organization"), strings.NewReader(string(requestPayload)))
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	if err != nil {
		return err
	}

	return nil
}
