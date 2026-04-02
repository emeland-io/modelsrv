package common

import "time"

type Version struct {
	Version        string
	AvailableFrom  *time.Time
	DeprecatedFrom *time.Time
	TerminatedFrom *time.Time
}

func (v Version) IsEqual(other Version) bool {
	if v.Version != other.Version {
		return false
	}

	if (v.AvailableFrom == nil) != (other.AvailableFrom == nil) {
		return false
	}
	if v.AvailableFrom != nil && !v.AvailableFrom.Equal(*other.AvailableFrom) {
		return false
	}

	if (v.DeprecatedFrom == nil) != (other.DeprecatedFrom == nil) {
		return false
	}
	if v.DeprecatedFrom != nil && !v.DeprecatedFrom.Equal(*other.DeprecatedFrom) {
		return false
	}

	if (v.TerminatedFrom == nil) != (other.TerminatedFrom == nil) {
		return false
	}
	if v.TerminatedFrom != nil && !v.TerminatedFrom.Equal(*other.TerminatedFrom) {
		return false
	}

	return true
}

type EntityVersion struct {
	Name    string
	Version string
}
