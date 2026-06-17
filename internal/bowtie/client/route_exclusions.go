package client

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// BowtieRouteExclusion is a split-tunnel rule: a collection of CIDRs that should
// be excluded from the Bowtie tunnel, optionally scoped to sites, WAN networks,
// and device/user attributes. Wire field names are kebab-case.
type BowtieRouteExclusion struct {
	ID                    string               `json:"id"`
	Name                  string               `json:"name"`
	CollectionID          string               `json:"collection-id"`
	Sites                 BowtieSiteDefinition `json:"sites"`
	ApplyStrategy         BowtieApplyStrategy  `json:"apply-strategy"`
	OnlyIfWANMatchesCIDRs []string             `json:"only-if-wan-matches-cidrs"`
	MatchOnlyDeviceOS     *string              `json:"match-only-device-os"`
	MatchOnlyDeviceType   *string              `json:"match-only-device-type"`
	MatchOnlyOwnership    *string              `json:"match-only-ownership"`
	MatchOnlyDeviceGroups []string             `json:"match-only-device-groups"`
	MatchOnlyUserGroups   []string             `json:"match-only-user-groups"`
	// Version is server-computed (a hash of the collection + sites) and read-only.
	Version string `json:"version,omitempty"`
}

// BowtieSiteDefinition is the adjacently tagged {type, value} site selector:
// {"type":"all"} or {"type":"specific","value":[site_id, ...]}.
type BowtieSiteDefinition struct {
	Type  string   `json:"type"`
	Value []string `json:"value,omitempty"`
}

// BowtieApplyStrategy is the adjacently tagged {type, value} rollout strategy:
// {"type":"always"} / {"type":"never"} /
// {"type":"percentage-user-match","value":N} /
// {"type":"percentage-device-match","value":N}.
type BowtieApplyStrategy struct {
	Type  string `json:"type"`
	Value *int   `json:"value,omitempty"`
}

func (c *Client) GetRouteExclusions() (map[string]BowtieRouteExclusion, error) {
	exclusions := map[string]BowtieRouteExclusion{}
	if err := c.getListJSON(&exclusions, "/route_exclusion/", "/route_exclusion"); err != nil {
		return nil, err
	}
	return exclusions, nil
}

func (c *Client) UpsertRouteExclusion(exclusion BowtieRouteExclusion) (BowtieRouteExclusion, error) {
	body, err := json.Marshal(exclusion)
	if err != nil {
		return BowtieRouteExclusion{}, err
	}

	var saved BowtieRouteExclusion
	if err := c.doJSONWithFallback(http.MethodPost, body, &saved, "/route_exclusion/", "/route_exclusion"); err != nil {
		return BowtieRouteExclusion{}, err
	}
	return saved, nil
}

func (c *Client) DeleteRouteExclusion(id string) error {
	req, err := http.NewRequest(http.MethodDelete, c.getHostURL(fmt.Sprintf("/route_exclusion/%s", id)), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}
