package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// BowtieCollection is a named, reusable set of network locations (IPs, CIDRs,
// DNS names, or other collections). Collections are referenced by resources
// (location type "collection"), route exclusions, and web filtering config.
type BowtieCollection struct {
	ID          string                            `json:"id"`
	Name        string                            `json:"name"`
	Description string                            `json:"description"`
	Source      *string                           `json:"source,omitempty"`
	SourceID    *string                           `json:"source_id,omitempty"`
	Members     map[string]BowtieCollectionMember `json:"members"`
}

// BowtieCollectionMember is one entry in a collection. Members may carry an
// optional expiry, after which the Controller drops them automatically.
type BowtieCollectionMember struct {
	ID       string                   `json:"id"`
	Name     string                   `json:"name"`
	Comment  string                   `json:"comment"`
	Expires  *string                  `json:"expires,omitempty"`
	Location BowtieCollectionLocation `json:"location"`
}

// BowtieCollectionLocation is the adjacently tagged {type, value} location used
// by collection members. type is one of ip, cidr, dns, or collection.
type BowtieCollectionLocation struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type collectionUpsert struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type addCollectionMembers struct {
	CollectionID string                   `json:"collection_id"`
	Members      []BowtieCollectionMember `json:"members"`
}

type removeCollectionMembers struct {
	CollectionID string   `json:"collection_id"`
	Members      []string `json:"members"`
}

func (c *Client) GetCollections() (map[string]BowtieCollection, error) {
	collections := map[string]BowtieCollection{}
	if err := c.getListJSON(&collections, "/collection/?with-members=true", "/collection?with-members=true"); err != nil {
		return nil, err
	}
	return collections, nil
}

func (c *Client) UpsertCollection(id, name, description string) error {
	payload, err := json.Marshal(collectionUpsert{
		ID:          id,
		Name:        name,
		Description: description,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.getHostURL("/collection/upsert"), bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}

func (c *Client) AddCollectionMembers(collectionID string, members []BowtieCollectionMember) error {
	if len(members) == 0 {
		return nil
	}

	payload, err := json.Marshal(addCollectionMembers{
		CollectionID: collectionID,
		Members:      members,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.getHostURL("/collection/addmember"), bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}

func (c *Client) RemoveCollectionMembers(collectionID string, memberIDs []string) error {
	if len(memberIDs) == 0 {
		return nil
	}

	for _, memberID := range memberIDs {
		payload, err := json.Marshal(removeCollectionMembers{
			CollectionID: collectionID,
			Members:      []string{memberID},
		})
		if err != nil {
			return err
		}

		req, err := http.NewRequest(http.MethodPost, c.getHostURL("/collection/removemember"), bytes.NewBuffer(payload))
		if err != nil {
			return err
		}

		if _, err = c.doRequest(req); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) DeleteCollection(id string) error {
	req, err := http.NewRequest(http.MethodDelete, c.getHostURL(fmt.Sprintf("/collection/%s", id)), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}
