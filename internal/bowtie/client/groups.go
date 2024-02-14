package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Group struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Users []string `json:"users,omitempty"`
}

type ModifyUserGroupPayload struct {
	GroupID string              `json:"group_id"`
	Users   []map[string]string `json:"users"`
}

type ModifyUserGroupResponse struct {
	Users map[string]bool `json:"users"`
}

type SetUserGroupMembershipPayload struct {
	Users []map[string]string `json:"users"`
}

func (c *Client) GetGroups() (map[string]Group, error) {
	groups, err := c.ListGroups()
	if err != nil {
		return nil, err
	}

	return groups, nil
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
	jsonErr := json.Unmarshal(body, &groups)
	if jsonErr != nil {
		return nil, jsonErr
	}

	return groups, nil
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
	jsonErr := json.Unmarshal(body, group)
	if jsonErr != nil {
		return "", jsonErr
	}

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
	var userIDPayloads []map[string]string = []map[string]string{}
	for _, userId := range userIDs {
		userIDPayloads = append(userIDPayloads, map[string]string{
			"id": userId,
		})
	}
	payload, err := json.Marshal(ModifyUserGroupPayload{
		GroupID: groupID,
		Users:   userIDPayloads,
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

func (c *Client) SetGroupMembership(groupID string, users []string) error {
	var userIDPayloads []map[string]string = []map[string]string{}
	for _, userId := range users {
		userIDPayloads = append(userIDPayloads, map[string]string{
			"id": userId,
		})
	}

	payload := SetUserGroupMembershipPayload{
		Users: userIDPayloads,
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
