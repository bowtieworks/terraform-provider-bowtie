package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type Group struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Users []string `json:"users,omitempty"`
}

type UserSpecifierSubId struct {
	Provider string `json:"provider"`
	SubID string `json:"subject_id"`
}

/// One of these must be set
type UserSpecifier struct {
	ID string `json:"id,omitempty"`
	Email string `json:"email,omitempty"`
	SubID UserSpecifierSubId `json:"sub,omitempty"`
}

type ModifyUserGroupPayload struct {
	GroupID string   `json:"group_id"`
	UserID  []UserSpecifier `json:"user_id"`
}

type ModifyUserGroupResponse struct {
	Updated map[string]bool `json:"updated"`
}

type SetUserGroupMembershipPayload struct {
	Users []UserSpecifier `json:"users"`
}

func (c *Client) GetGroup(id string) (*Group, error) {
	groups, err := c.ListGroups()
	if err != nil {
		return nil, err
	}

	group, ok := groups[id]
	if !ok {
		return nil, fmt.Errorf("failed to find group with id: %s", id)
	}
	return &group, nil
}

func (c *Client) ListGroups() (map[string]Group, error) {
	req, err := http.NewRequest("GET", c.getHostURL("/group"), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var groups map[string]Group = map[string]Group{}
	json.Unmarshal(body, &groups)

	return groups, nil
}

func (c *Client) CreateGroup(name string) (string, error) {
	return c.UpsertGroup(uuid.NewString(), name)
}

func (c *Client) UpsertGroup(id, name string) (string, error) {
	groupRequest := Group{
		Name: name,
		ID:   id,
	}

	requestBody, err := json.Marshal(groupRequest)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", c.getHostURL("/group/upsert"), strings.NewReader(string(requestBody)))
	if err != nil {
		return "", err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return "", err
	}

	var group *Group = &Group{}
	json.Unmarshal(body, group)

	return id, nil
}

func (c *Client) ListUsersInGroup(id string) (*Group, error) {
	req, err := http.NewRequest("GET", c.getHostURL(fmt.Sprintf("/group/%s/list", id)), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var group *Group = &Group{}
	err = json.Unmarshal(body, group)
	if err != nil {
		return nil, err
	}

	return group, nil
}

func (c *Client) AddUserToGroup(groupID string, userIDs []string) (*ModifyUserGroupResponse, error) {
	return c.modifyUserGroup("addusers", groupID, userIDs)
}

func (c *Client) RemoveUserFromGroup(groupID string, userIDs []string) (*ModifyUserGroupResponse, error) {
	return c.modifyUserGroup("removeusers", groupID, userIDs)
}

func (c *Client) modifyUserGroup(action, groupID string, userIDs []string) (*ModifyUserGroupResponse, error) {
	
	payload, err := json.Marshal(ModifyUserGroupPayload{
		GroupID: groupID,
		UserID:  userIDs,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.getHostURL(fmt.Sprintf("/group/%s", action)), strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var response *ModifyUserGroupResponse = &ModifyUserGroupResponse{}
	err = json.Unmarshal(body, response)

	return response, err
}

func (c *Client) DeleteGroup(groupID string) error {
	req, err := http.NewRequest(http.MethodDelete, c.getHostURL(fmt.Sprintf("/group/%s", groupID)), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}

func (c *Client) SetGroupMembership(groupID string, users []UserSpecifier) error {
	payload := SetUserGroupMembershipPayload{
		Users: users,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.getHostURL(fmt.Sprintf("/group/%s/set_membership", groupID)), bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}
