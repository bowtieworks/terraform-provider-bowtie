package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func (c *Client) ListSites() ([]Site, error) {
	org, err := c.GetOrganization()
	if err != nil {
		return nil, err
	}

	return org.Sites, nil
}

func (c *Client) GetSites() ([]Site, error) {
	org, err := c.GetOrganization()
	if err != nil {
		return nil, err
	}

	return org.Sites, nil
}

func FindSite(siteID string, sites []Site) (*Site, error) {
	for _, site := range sites {
		if site.ID == siteID {
			return &site, nil
		}
	}

	return nil, fmt.Errorf("site not found: %s", siteID)
}

type SiteUpsertPayload struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *Client) CreateSite(id string, name string) error {
	err := c.UpsertSite(id, name)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) UpsertSite(id, name string) error {
	payload := SiteUpsertPayload{
		ID:   id,
		Name: name,
	}

	requestPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.getHostURL("/site"), strings.NewReader(string(requestPayload)))
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	if err != nil {
		return err
	}

	return nil
}

type siteRangePayload struct {
	ID          string `json:"id"`
	SiteID      string `json:"site_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Range       string `json:"range"`
	IsV4        bool   `json:"is_v4"`
	IsV6        bool   `json:"is_v6"`
	Weight      int64  `json:"weight"`
	Metric      int64  `json:"metric"`
}

func (c *Client) DeleteSite(siteID string) error {
	req, err := http.NewRequest(http.MethodDelete, c.getHostURL(fmt.Sprintf("/site/%s", siteID)), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}

func (c *Client) UpsertSiteRange(siteID, id, name, description, cidr string, isV4, isV6 bool, weight, metric int64) error {
	payload := siteRangePayload{
		ID:          id,
		SiteID:      siteID,
		Name:        name,
		Description: description,
		Range:       cidr,
		IsV4:        isV4,
		IsV6:        isV6,
		Weight:      weight,
		Metric:      metric,
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.getHostURL(fmt.Sprintf("/site/%s/range", siteID)), strings.NewReader(string(requestBody)))
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}

func (c *Client) DeleteSiteRange(siteID, id string) error {
	req, err := http.NewRequest(http.MethodDelete, c.getHostURL(fmt.Sprintf("/site/%s/range/%s", siteID, id)), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)

	return err
}

func (c *Client) FindSiteRange(site *Site, id string) (*RoutableRange, error) {
	// return which range type it is
	for _, info := range site.RoutableRangesV4 {
		if info.ID == id {
			info.ISV4 = true
			return &info, nil
		}
	}

	for _, info := range site.RouteRangesV6 {
		if info.ID == id {
			info.ISV6 = true
			return &info, nil
		}
	}

	return nil, fmt.Errorf("routable range not found in site: range: %s", id)
}
