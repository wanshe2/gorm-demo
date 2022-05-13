package model

type Method interface {
	// FindMaxVersionCount query count of MAX(version)
	//
	// sql(SELECT COUNT(*) FROM users GROUP BY version ORDER BY version DESC LIMIT 1)
	FindMaxVersionCount() (uint8, error)
}
