package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// BowtiePolicy mirrors the policy engine's rule document. A policy grants or
// denies a source (described by a predicate) access to a destination resource
// group.
type BowtiePolicy struct {
	ID     string             `json:"id"`
	Source BowtiePolicySource `json:"source"`
	Dest   string             `json:"dest"`
	Action string             `json:"action"`
	// Order is a pointer so that an unset order is omitted from the request,
	// letting the Controller assign the next position. Sending an explicit
	// zero pins the policy to the top of the list.
	Order  *int64 `json:"order,omitempty"`
	Status string `json:"status,omitempty"`
}

// BowtiePolicySource pairs a stable identifier with the predicate that decides
// whether traffic matches the rule.
type BowtiePolicySource struct {
	ID        string          `json:"id"`
	Predicate BowtiePredicate `json:"predicate"`
}

// BowtiePredicate is the Go representation of the server's SourcePredicate enum.
//
// On the wire the predicate is either a bare string ("Always",
// "AuthenticatedUser") or a single-key object tagging the variant
// ({"User": "<uuid>"}, {"And": [<source>, ...]}, ...). Exactly one field is
// populated for a valid predicate; the And/Or/Nor variants nest further
// sources, so the type is recursive and supports arbitrary depth.
type BowtiePredicate struct {
	Always            bool
	AuthenticatedUser bool
	User              string
	Device            string
	InUserGroup       string
	InDeviceGroup     string
	And               []BowtiePolicySource
	Or                []BowtiePolicySource
	Nor               []BowtiePolicySource
}

const (
	predicateAlways            = "Always"
	predicateAuthenticatedUser = "AuthenticatedUser"
	predicateUser              = "User"
	predicateDevice            = "Device"
	predicateInUserGroup       = "InUserGroup"
	predicateInDeviceGroup     = "InDeviceGroup"
	predicateAnd               = "And"
	predicateOr                = "Or"
	predicateNor               = "Nor"
)

func (p BowtiePredicate) MarshalJSON() ([]byte, error) {
	switch {
	case p.Always:
		return json.Marshal(predicateAlways)
	case p.AuthenticatedUser:
		return json.Marshal(predicateAuthenticatedUser)
	case p.User != "":
		return json.Marshal(map[string]string{predicateUser: p.User})
	case p.Device != "":
		return json.Marshal(map[string]string{predicateDevice: p.Device})
	case p.InUserGroup != "":
		return json.Marshal(map[string]string{predicateInUserGroup: p.InUserGroup})
	case p.InDeviceGroup != "":
		return json.Marshal(map[string]string{predicateInDeviceGroup: p.InDeviceGroup})
	case p.And != nil:
		return json.Marshal(map[string][]BowtiePolicySource{predicateAnd: p.And})
	case p.Or != nil:
		return json.Marshal(map[string][]BowtiePolicySource{predicateOr: p.Or})
	case p.Nor != nil:
		return json.Marshal(map[string][]BowtiePolicySource{predicateNor: p.Nor})
	default:
		return nil, fmt.Errorf("policy source predicate has no variant set")
	}
}

func (p *BowtiePredicate) UnmarshalJSON(data []byte) error {
	// The string variants (Always, AuthenticatedUser) arrive as bare strings.
	var tag string
	if err := json.Unmarshal(data, &tag); err == nil {
		switch tag {
		case predicateAlways:
			p.Always = true
		case predicateAuthenticatedUser:
			p.AuthenticatedUser = true
		default:
			return fmt.Errorf("unknown string predicate %q", tag)
		}
		return nil
	}

	// Every remaining variant is a single-key object.
	var tagged map[string]json.RawMessage
	if err := json.Unmarshal(data, &tagged); err != nil {
		return fmt.Errorf("predicate is neither a known string nor an object: %w", err)
	}
	if len(tagged) != 1 {
		return fmt.Errorf("expected exactly one predicate variant, found %d", len(tagged))
	}

	for variant, raw := range tagged {
		switch variant {
		case predicateUser:
			return json.Unmarshal(raw, &p.User)
		case predicateDevice:
			return json.Unmarshal(raw, &p.Device)
		case predicateInUserGroup:
			return json.Unmarshal(raw, &p.InUserGroup)
		case predicateInDeviceGroup:
			return json.Unmarshal(raw, &p.InDeviceGroup)
		case predicateAnd:
			return json.Unmarshal(raw, &p.And)
		case predicateOr:
			return json.Unmarshal(raw, &p.Or)
		case predicateNor:
			return json.Unmarshal(raw, &p.Nor)
		default:
			return fmt.Errorf("unknown predicate variant %q", variant)
		}
	}

	return nil
}

// UpsertPolicy creates or replaces a policy. The server assigns an order when
// one is not supplied, so the persisted policy is returned to the caller.
func (c *Client) UpsertPolicy(policy BowtiePolicy) (BowtiePolicy, error) {
	body, err := json.Marshal(policy)
	if err != nil {
		return BowtiePolicy{}, err
	}

	req, err := http.NewRequest(http.MethodPost, c.getHostURL("/policy/upsert_policy"), bytes.NewBuffer(body))
	if err != nil {
		return BowtiePolicy{}, err
	}

	responseBody, err := c.doRequest(req)
	if err != nil {
		return BowtiePolicy{}, err
	}

	var saved BowtiePolicy
	if err := json.Unmarshal(responseBody, &saved); err != nil {
		return BowtiePolicy{}, err
	}

	return saved, nil
}
