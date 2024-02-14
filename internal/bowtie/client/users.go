package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type BowtieUser struct {
	ID                string `json:"id"`
	Name              string `json:"name,omitempty"`
	Email             string `json:"email,omitempty"`
	AuthzDevices      *bool  `json:"authz_devices,omitempty"`
	AuthzPolicies     *bool  `json:"authz_policies,omitempty"`
	AuthzControlPlane *bool  `json:"authz_control_plane,omitempty"`
	AuthzUsers        *bool  `json:"authz_users,omitempty"`
	Status            string `json:"status,omitempty"`
	Role              string `json:"role,omitempty"`
}

func (c *Client) GetUsers() (map[string]BowtieUser, error) {
	req, err := http.NewRequest(http.MethodGet, c.getHostURL("/users"), nil)
	if err != nil {
		return nil, err
	}

	responseBody, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var users map[string]BowtieUser = map[string]BowtieUser{}
	err = json.Unmarshal(responseBody, &users)
	return users, err
}

func (c *Client) GetUserByEmail(ctx context.Context, email string) (BowtieUser, error) {
	users, err := c.GetUsers()
	if err != nil {
		return BowtieUser{}, err
	}

	for _, user := range users {
		if user.Email == strings.TrimSpace(email) {
			return user, nil
		}
	}

	return BowtieUser{}, fmt.Errorf("user not found")
}

func (c *Client) GetUser(id string) (BowtieUser, error) {
	req, err := http.NewRequest(http.MethodGet, c.getHostURL(fmt.Sprintf("/user/%s", id)), nil)
	if err != nil {
		return BowtieUser{}, nil
	}

	body, err := c.doRequest(req)
	if err != nil {
		return BowtieUser{}, err
	}

	var user BowtieUser = BowtieUser{}
	err = json.Unmarshal(body, &user)
	return user, err
}

func (c *Client) DeleteUser(ctx context.Context, id string) error {
	req, err := http.NewRequest(http.MethodDelete, c.getHostURL(fmt.Sprintf("/user/%s", id)), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}

func (c *Client) DisableUser(ctx context.Context, id string) error {
	payload := BowtieUser{
		Status: "Disabled",
		ID:     id,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.getHostURL("/user/upsert"), bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}

func (c *Client) UpsertUser(id, name, email, role string, authzPolicies, authzUsers, authzControlPlane, authzDevices, enabled bool) (string, error) {
	payload := BowtieUser{
		ID:                id,
		Name:              name,
		Email:             email,
		AuthzDevices:      &authzDevices,
		AuthzPolicies:     &authzPolicies,
		AuthzControlPlane: &authzControlPlane,
		AuthzUsers:        &authzUsers,
		Role:              role,
		Status:            "Active",
	}

	if !enabled {
		payload.Status = "Disabled"
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, c.getHostURL("/user/upsert"), bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	_, err = c.doRequest(req)
	return id, err
}
