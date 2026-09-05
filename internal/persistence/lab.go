package persistence

import "fmt"

// LabID is a 16-byte laboratory identifier encoded as 32 lowercase hex digits.
type LabID string

func (id LabID) Validate() error {
	if len(id) != 32 {
		return fmt.Errorf("invalid laboratory ID")
	}
	for _, c := range id {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return fmt.Errorf("invalid laboratory ID")
		}
	}
	return nil
}

// LabRepository binds every gameplay operation to one immutable laboratory ID.
// Its Store owns the database connection and must outlive this repository.
type LabRepository struct {
	store *Store
	labID LabID
}

func (s *Store) ForLab(id LabID) (*LabRepository, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("bind laboratory: nil store")
	}
	if err := id.Validate(); err != nil {
		return nil, err
	}
	return &LabRepository{store: s, labID: id}, nil
}

func (r *LabRepository) valid() bool {
	return r != nil && r.store != nil && r.store.db != nil && r.labID.Validate() == nil
}
