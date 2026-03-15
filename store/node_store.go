package store

import (
	"github.com/goozt/seashell/model"
)

const (
	prefixNode       = "node:"
	prefixNodeReq    = "nodereq:"
)

// SaveNode persists a node record.
func (d *DB) SaveNode(n *model.Node) error {
	return d.set(prefixNode+n.ID, n)
}

// GetNodeByID retrieves a node by ID.
func (d *DB) GetNodeByID(id string) (*model.Node, error) {
	var n model.Node
	if err := d.get(prefixNode+id, &n); err != nil {
		return nil, err
	}
	return &n, nil
}

// ListNodes returns all node records.
func (d *DB) ListNodes() ([]*model.Node, error) {
	var all []*model.Node
	err := d.iterPrefix(prefixNode, func(val []byte) error {
		var n model.Node
		if err := jsonUnmarshal(val, &n); err != nil {
			return err
		}
		all = append(all, &n)
		return nil
	})
	return all, err
}

// ListActiveNodes returns nodes with status=active.
func (d *DB) ListActiveNodes() ([]*model.Node, error) {
	all, err := d.ListNodes()
	if err != nil {
		return nil, err
	}
	var result []*model.Node
	for _, n := range all {
		if n.Status == model.NodeStatusActive {
			result = append(result, n)
		}
	}
	return result, nil
}

// DeleteNode removes a node record.
func (d *DB) DeleteNode(id string) error {
	return d.del(prefixNode + id)
}

// SaveNodeJoinRequest persists a node join request.
func (d *DB) SaveNodeJoinRequest(r *model.NodeJoinRequest) error {
	return d.set(prefixNodeReq+r.ID, r)
}

// GetNodeJoinRequestByID retrieves a node join request by ID.
func (d *DB) GetNodeJoinRequestByID(id string) (*model.NodeJoinRequest, error) {
	var r model.NodeJoinRequest
	if err := d.get(prefixNodeReq+id, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// ListNodeJoinRequests returns all node join requests.
func (d *DB) ListNodeJoinRequests() ([]*model.NodeJoinRequest, error) {
	var all []*model.NodeJoinRequest
	err := d.iterPrefix(prefixNodeReq, func(val []byte) error {
		var r model.NodeJoinRequest
		if err := jsonUnmarshal(val, &r); err != nil {
			return err
		}
		all = append(all, &r)
		return nil
	})
	return all, err
}

// ListNodesByTier returns all nodes with the given NodeTier value.
// Nodes with empty NodeTier are treated as "branch".
func (d *DB) ListNodesByTier(tier string) ([]*model.Node, error) {
	all, err := d.ListNodes()
	if err != nil {
		return nil, err
	}
	var result []*model.Node
	for _, n := range all {
		nodeTier := n.NodeTier
		if nodeTier == "" {
			nodeTier = model.NodeTierBranch
		}
		if nodeTier == tier {
			result = append(result, n)
		}
	}
	return result, nil
}

// ListNodesByParent returns all nodes whose ParentNodeID matches the given ID.
func (d *DB) ListNodesByParent(parentNodeID string) ([]*model.Node, error) {
	all, err := d.ListNodes()
	if err != nil {
		return nil, err
	}
	var result []*model.Node
	for _, n := range all {
		if n.ParentNodeID == parentNodeID {
			result = append(result, n)
		}
	}
	return result, nil
}

// ListPendingNodeJoinRequests returns node join requests with status=pending.
func (d *DB) ListPendingNodeJoinRequests() ([]*model.NodeJoinRequest, error) {
	all, err := d.ListNodeJoinRequests()
	if err != nil {
		return nil, err
	}
	var result []*model.NodeJoinRequest
	for _, r := range all {
		if r.Status == "pending" {
			result = append(result, r)
		}
	}
	return result, nil
}
