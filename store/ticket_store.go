package store

import "github.com/goozt/seashell/model"

const prefixTicket = "ticket:"

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
		result = append(result, &t)
		return nil
	})
	return result, err
}
