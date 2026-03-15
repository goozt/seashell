package store

import "github.com/goozt/seashell/model"

const prefixTicket = "ticket:"

// enrichReplyUsernames resolves AuthorID → AuthorUsername for every reply
// that is missing a proper username (empty or equal to the UUID).
func (d *DB) enrichReplyUsernames(t *model.Ticket) {
	for i := range t.Replies {
		r := &t.Replies[i]
		if r.AuthorUsername == "" || r.AuthorUsername == r.AuthorID {
			if u, err := d.GetUserByID(r.AuthorID); err == nil {
				r.AuthorUsername = u.Username
			}
		}
	}
}

// SaveTicket persists a ticket (create or update).
func (d *DB) SaveTicket(t *model.Ticket) error {
	return d.set(prefixTicket+t.ID, t)
}

// GetTicketByID retrieves a ticket by ID.
func (d *DB) GetTicketByID(id string) (*model.Ticket, error) {
	var t model.Ticket
	if err := d.get(prefixTicket+id, &t); err != nil {
		return nil, err
	}
	d.enrichReplyUsernames(&t)
	return &t, nil
}

// ListTicketsByAuthority returns all tickets for a given authority.
func (d *DB) ListTicketsByAuthority(authorityID string) ([]*model.Ticket, error) {
	var result []*model.Ticket
	err := d.iterPrefix(prefixTicket, func(val []byte) error {
		var t model.Ticket
		if err := jsonUnmarshal(val, &t); err != nil {
			return err
		}
		if t.AuthorityID == authorityID {
			d.enrichReplyUsernames(&t)
			result = append(result, &t)
		}
		return nil
	})
	return result, err
}

// ListTicketsByUser returns all tickets created by a specific user.
func (d *DB) ListTicketsByUser(userID string) ([]*model.Ticket, error) {
	var result []*model.Ticket
	err := d.iterPrefix(prefixTicket, func(val []byte) error {
		var t model.Ticket
		if err := jsonUnmarshal(val, &t); err != nil {
			return err
		}
		if t.CreatedBy == userID {
			d.enrichReplyUsernames(&t)
			result = append(result, &t)
		}
		return nil
	})
	return result, err
}

// ListTicketsByLevel returns tickets whose EscalatedTo matches the given level.
// Pass model.TicketLevelNode (or "") to get tickets at node level (not yet escalated).
func (d *DB) ListTicketsByLevel(level string) ([]*model.Ticket, error) {
	var result []*model.Ticket
	err := d.iterPrefix(prefixTicket, func(val []byte) error {
		var t model.Ticket
		if err := jsonUnmarshal(val, &t); err != nil {
			return err
		}
		effective := t.EscalatedTo
		if effective == "" {
			effective = model.TicketLevelNode
		}
		if effective == level {
			d.enrichReplyUsernames(&t)
			result = append(result, &t)
		}
		return nil
	})
	return result, err
}

// ListAllTickets returns every ticket in the system (admin view).
func (d *DB) ListAllTickets() ([]*model.Ticket, error) {
	var result []*model.Ticket
	err := d.iterPrefix(prefixTicket, func(val []byte) error {
		var t model.Ticket
		if err := jsonUnmarshal(val, &t); err != nil {
			return err
		}
		d.enrichReplyUsernames(&t)
		result = append(result, &t)
		return nil
	})
	return result, err
}
