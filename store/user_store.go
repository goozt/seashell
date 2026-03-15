package store

import (
	"fmt"

	"github.com/goozt/seashell/model"
)

const (
	prefixUser        = "user:"
	prefixUserEmail   = "idx:user:email:"
	prefixUserUsername = "idx:user:username:"
)

// SaveUser persists a user record and updates email/username indexes.
func (d *DB) SaveUser(u *model.User) error {
	if err := d.set(prefixUser+u.ID, u); err != nil {
		return err
	}
	if err := d.set(prefixUserEmail+u.Email, u.ID); err != nil {
		return err
	}
	return d.set(prefixUserUsername+u.Username, u.ID)
}

// GetUserByID retrieves a user by their UUID.
func (d *DB) GetUserByID(id string) (*model.User, error) {
	var u model.User
	if err := d.get(prefixUser+id, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByEmail looks up a user by email.
func (d *DB) GetUserByEmail(email string) (*model.User, error) {
	var id string
	if err := d.get(prefixUserEmail+email, &id); err != nil {
		return nil, err
	}
	return d.GetUserByID(id)
}

// GetUserByUsername looks up a user by username.
func (d *DB) GetUserByUsername(username string) (*model.User, error) {
	var id string
	if err := d.get(prefixUserUsername+username, &id); err != nil {
		return nil, err
	}
	return d.GetUserByID(id)
}

// EmailExists returns true if the email is already registered.
func (d *DB) EmailExists(email string) bool {
	var id string
	return d.get(prefixUserEmail+email, &id) == nil
}

// UsernameExists returns true if the username is already taken.
func (d *DB) UsernameExists(username string) bool {
	var id string
	return d.get(prefixUserUsername+username, &id) == nil
}

// ListUsers returns all users (paginated). page is 1-based.
func (d *DB) ListUsers(page, limit int) ([]*model.User, int, error) {
	var all []*model.User
	err := d.iterPrefix(prefixUser, func(val []byte) error {
		var u model.User
		if err := jsonUnmarshal(val, &u); err != nil {
			return err
		}
		all = append(all, &u)
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	total := len(all)
	start, end := paginate(total, page, limit)
	return all[start:end], total, nil
}

// ListUsersByAuthorityID returns all users belonging to the given authority.
func (d *DB) ListUsersByAuthorityID(authorityID string) ([]*model.User, error) {
	var result []*model.User
	err := d.iterPrefix(prefixUser, func(val []byte) error {
		var u model.User
		if err := jsonUnmarshal(val, &u); err != nil {
			return err
		}
		if u.AuthorityID == authorityID {
			result = append(result, &u)
		}
		return nil
	})
	return result, err
}

// CountAdmins returns the number of users with role admin or superadmin.
func (d *DB) CountAdmins() (int, error) {
	count := 0
	err := d.iterPrefix(prefixUser, func(val []byte) error {
		var u model.User
		if err := jsonUnmarshal(val, &u); err != nil {
			return err
		}
		if u.Role == model.RoleAdmin || u.Role == model.RoleSuperAdmin {
			count++
		}
		return nil
	})
	return count, err
}

// ListAdmins returns all users with role=admin.
func (d *DB) ListAdmins() ([]*model.User, error) {
	var result []*model.User
	err := d.iterPrefix(prefixUser, func(val []byte) error {
		var u model.User
		if err := jsonUnmarshal(val, &u); err != nil {
			return err
		}
		if u.Role == model.RoleAdmin {
			result = append(result, &u)
		}
		return nil
	})
	return result, err
}

// SuperAdminExists returns true if any superadmin account exists.
func (d *DB) SuperAdminExists() bool {
	found := false
	_ = d.iterPrefix(prefixUser, func(val []byte) error {
		var u model.User
		if err := jsonUnmarshal(val, &u); err != nil {
			return err
		}
		if u.Role == model.RoleSuperAdmin {
			found = true
		}
		return nil
	})
	return found
}

// StoreWalletPrivKey stores the raw D bytes of a user's ECDSA private key.
func (d *DB) StoreWalletPrivKey(userID string, dBytes []byte) error {
	return d.setRaw(fmt.Sprintf("wallet_privkey:%s", userID), dBytes)
}

// GetWalletPrivKey retrieves the raw D bytes of a user's ECDSA private key.
func (d *DB) GetWalletPrivKey(userID string) ([]byte, error) {
	return d.getRaw(fmt.Sprintf("wallet_privkey:%s", userID))
}
