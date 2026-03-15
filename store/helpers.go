package store

import "encoding/json"

// jsonUnmarshal is a package-local alias used by store methods.
func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// paginate returns safe start/end slice indices for the given page and limit.
// page is 1-based.
func paginate(total, page, limit int) (start, end int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	start = (page - 1) * limit
	if start >= total {
		return total, total
	}
	end = start + limit
	if end > total {
		end = total
	}
	return start, end
}
