package search

import (
	"database/sql"

	"github.com/lib/pq"

	"ispilolite/internal/models"
)

// sqlNoRows aliases sql.ErrNoRows for the fallback repository's best-effort
// suggestion lookups.
var sqlNoRows = sql.ErrNoRows

// rowScanner is satisfied by *sql.Rows, letting scanTechRow serve both the
// plain and geo-distance technician queries.
type rowScanner interface {
	Scan(dest ...interface{}) error
}

// scanTechRow decodes a technician row. When withDistance is true the query is
// expected to select a trailing distance_km column.
func scanTechRow(rows rowScanner, withDistance bool) (*models.Technician, error) {
	var (
		t      models.Technician
		roles  pq.StringArray
		skills pq.StringArray
	)
	dest := []interface{}{
		&t.ID, &t.UserID, &t.Name, &t.AvatarURL, &t.ISPID, &t.ISPName,
		&t.County, &t.SubCounty, &t.Village, &t.Rating, &t.JobsDone, &t.IsAvailable,
		&roles, &skills,
	}
	if withDistance {
		dest = append(dest, &t.Distance)
	}
	if err := rows.Scan(dest...); err != nil {
		return nil, err
	}
	t.Roles = roles
	t.Skills = skills
	return &t, nil
}
