package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type PoliciesEndpointResponse struct {
	Policies       map[string]BowtiePolicy        `json:"policies"`
	ResourceGroups map[string]BowtieResourceGroup `json:"resource_groups"`
	Resources      map[string]BowtieResource      `json:"resources"`
}

type BowtiePolicy struct {
	ID string `json:"id"`
	// Source BowtiePolicySource `json:"source"`
	// Dest   string             `json:"dest"`
	// Action string             `json:"action"`
}

type BowtiePolicySource struct {
	ID        string `json:"id"`
	Predicate string `json:"predicate"`
}

type BowtieResourceGroup struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Inherited []string `json:"inherited"`
	Resources []string `json:"resources"`
}

type BowtieResource struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Protocol string                 `json:"protocol"`
	Location BowtieResourceLocation `json:"location"`
	Ports    BowtieResourcePorts    `json:"ports"`
}

// TODO: This can be collapsed down into the tagged-only variant once
// the old style API is sunset.
type BowtieResourceLocation struct {
	Untagged *BowtieResourceLocationUntagged
	Tagged   *BowtieResourceLocationTagged
}

func (rl *BowtieResourceLocation) UnmarshalJSON(data []byte) error {
	// First, try unmarshaling into BowtieResourceLocationTagged
	//
	// Unmarshal ordering is important here since the untagged version
	// has all-optional fields.
	if err := json.Unmarshal(data, &rl.Tagged); err == nil {
		return nil
	}

	// Then, try unmarshaling into BowtieResourceLocationUntagged
	if err := json.Unmarshal(data, &rl.Untagged); err == nil {
		return nil
	}

	// If neither unmarshaling works, return an error
	return errors.New("failed to unmarshal API response for location")
}

func (rl BowtieResourceLocation) MarshalJSON() ([]byte, error) {
	// Determine which version to marshal based on which field is non-nil
	if rl.Untagged != nil {
		return json.Marshal(rl.Untagged)
	} else if rl.Tagged != nil {
		return json.Marshal(rl.Tagged)
	}
	return nil, errors.New("no data to marshal")
}

type BowtieResourceLocationUntagged struct {
	IP   string `json:"ip,omitempty"`
	CIDR string `json:"cidr,omitempty"`
	DNS  string `json:"dns,omitempty"`
}

type BowtieResourceLocationTagged struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type BowtieResourcePorts struct {
	Range      []int64                       `json:"range,omitempty"`
	Collection *BowtieResourcePortCollection `json:"collection,omitempty"`
}

type BowtieResourcePortCollection struct {
	Ports []int64 `json:"ports,omitempty"`
}

func (c *Client) UpsertResource(id, name, protocol string, location BowtieResourceLocation, portRange, portCollection []int64) (BowtieResource, error) {
	payload := BowtieResource{
		ID:       id,
		Name:     name,
		Protocol: protocol,
		Location: location,
		Ports:    BowtieResourcePorts{},
	}

	if len(portRange) > 0 {
		payload.Ports.Range = portRange
	} else {
		payload.Ports.Collection = &BowtieResourcePortCollection{
			Ports: portCollection,
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return BowtieResource{}, err
	}

	if err != nil {
		return BowtieResource{}, err
	}

	req, err := http.NewRequest(http.MethodPost, c.getHostURL("/policy/upsert_resource"), bytes.NewBuffer(body))
	if err != nil {
		return BowtieResource{}, err
	}

	responsePayload, err := c.doRequest(req)
	if err != nil {
		return BowtieResource{}, err
	}

	var resource BowtieResource = BowtieResource{}
	err = json.Unmarshal(responsePayload, &resource)
	if err != nil {
		return BowtieResource{}, err
	}

	return resource, nil
}

func (c *Client) GetPoliciesAndResources() (*PoliciesEndpointResponse, error) {
	req, err := http.NewRequest(http.MethodGet, c.getHostURL("/policy"), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var policy *PoliciesEndpointResponse = &PoliciesEndpointResponse{}
	err = json.Unmarshal(body, &policy)
	return policy, err
}

func (c *Client) GetPolicy(id string) (BowtiePolicy, error) {
	policyInfo, err := c.GetPoliciesAndResources()
	if err != nil {
		return BowtiePolicy{}, err
	}

	policy, ok := policyInfo.Policies[id]
	if !ok {
		return BowtiePolicy{}, fmt.Errorf("policy not found")
	}

	return policy, nil
}

func (c *Client) GetResourceGroups() (map[string]BowtieResourceGroup, error) {
	rp, err := c.GetPoliciesAndResources()
	if err != nil {
		return make(map[string]BowtieResourceGroup), nil
	}

	return rp.ResourceGroups, nil
}

func (c *Client) GetResources() (map[string]BowtieResource, error) {
	rp, err := c.GetPoliciesAndResources()
	if err != nil {
		return make(map[string]BowtieResource), err
	}

	return rp.Resources, nil
}

func (c *Client) DeletePolicy(id string) error {
	req, err := http.NewRequest(http.MethodDelete, c.getHostURL(fmt.Sprintf("/policy/%s", id)), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}

func (c *Client) DeleteResource(id string) error {
	req, err := http.NewRequest(http.MethodDelete, c.getHostURL(fmt.Sprintf("/policy/resource/%s", id)), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}

func (c *Client) UpsertResourceGroup(id, name string, resources, resource_groups []string) error {
	payload := BowtieResourceGroup{
		ID:        id,
		Name:      name,
		Resources: resources,
		Inherited: resource_groups,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.getHostURL("/policy/upsert_resource_group"), bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}

func (c *Client) DeleteResourceGroup(id string) error {
	req, err := http.NewRequest(http.MethodDelete, c.getHostURL(fmt.Sprintf("/policy/resource_group/%s", id)), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}
