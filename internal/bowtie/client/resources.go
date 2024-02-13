package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
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

type BowtieResourceLocation struct {
	IP   string `json:"ip,omitempty"`
	CIDR string `json:"cidr,omitempty"`
	DNS  string `json:"dns,omitempty"`
}

type BowtieResourcePorts struct {
	Range      []int64                       `json:"range,omitempty"`
	Collection *BowtieResourcePortCollection `json:"collection,omitempty"`
}

type BowtieResourcePortCollection struct {
	Ports []int64 `json:"ports,omitempty"`
}

func (c *Client) UpsertResource(id, name, protocol, ip, cidr, dns string, portRange, portCollection []int64) (BowtieResource, error) {
	payload := BowtieResource{
		ID:       id,
		Name:     name,
		Protocol: protocol,
		Location: BowtieResourceLocation{
			IP:   ip,
			CIDR: cidr,
			DNS:  dns,
		},
		Ports: BowtieResourcePorts{},
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

func (c *Client) GetResourceGroup(id string) (BowtieResourceGroup, error) {
	rp, err := c.GetPoliciesAndResources()
	if err != nil {
		return BowtieResourceGroup{}, nil
	}

	for _, val := range rp.ResourceGroups {
		if val.ID == id {
			return val, nil
		}
	}

	return BowtieResourceGroup{}, fmt.Errorf("resource_group not found")
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

func (c *Client) CreateResourceGroup(ctx context.Context, name string, resources, resource_groups []string) (string, error) {
	id := uuid.NewString()
	return id, c.UpsertResourceGroup(ctx, id, name, resources, resource_groups)
}

func (c *Client) UpsertResourceGroup(ctx context.Context, id, name string, resources, resource_groups []string) error {
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
