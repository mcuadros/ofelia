package core

import "github.com/gohugoio/hashstructure"

// A promoted BareJob.Hash only sees the embedded fields, so each job hashes its own settings.
func (j *ExecJob) Hash() uint64 {
	hash, _ := hashstructure.Hash(j, nil)
	return hash
}

func (j *RunJob) Hash() uint64 {
	hash, _ := hashstructure.Hash(j, nil)
	return hash
}

func (j *RunServiceJob) Hash() uint64 {
	hash, _ := hashstructure.Hash(j, nil)
	return hash
}

func (j *LocalJob) Hash() uint64 {
	hash, _ := hashstructure.Hash(j, nil)
	return hash
}
