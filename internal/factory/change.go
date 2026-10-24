package factory

import (
	"context"
	"strconv"
	"strings"
)

const maxChangePatch = 200 << 10

// JobChange is what a reviewer reads: the files the candidate changed against the base
// it started from, and the patch.
type JobChange struct {
	JobID             JobID         `json:"jobId"`
	BaseRevision      string        `json:"baseRevision"`
	CandidateRevision string        `json:"candidateRevision"`
	Files             []ChangedFile `json:"files"`
	Patch             string        `json:"patch"`
	Truncated         bool          `json:"truncated"`
}

type ChangedFile struct {
	Path    string `json:"path"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
}

func (s *Service) Change(ctx context.Context, actor Principal, jobID JobID) (JobChange, error) {
	job, err := s.GetJob(ctx, actor, jobID)
	if err != nil {
		return JobChange{}, err
	}
	if job.CandidateRevision == "" {
		return JobChange{}, fault(CodeNotReady, "the job has no candidate yet", nil)
	}
	project, err := readProject(ctx, s.state.db, job.ProjectID)
	if err != nil {
		return JobChange{}, err
	}
	change := JobChange{JobID: job.ID, BaseRevision: job.Admission.BaseRevision, CandidateRevision: job.CandidateRevision, Files: []ChangedFile{}}
	span := job.Admission.BaseRevision + ".." + job.CandidateRevision
	stat, err := s.git(ctx, project.RepositoryPath, "diff", "--numstat", "--no-renames", span)
	if err != nil {
		return JobChange{}, fault(CodeInternal, "read the change", err)
	}
	for _, line := range strings.Split(stat, "\n") {
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) != 3 {
			continue
		}
		added, _ := strconv.Atoi(fields[0])
		removed, _ := strconv.Atoi(fields[1])
		change.Files = append(change.Files, ChangedFile{Path: fields[2], Added: added, Removed: removed})
	}
	patch, err := s.git(ctx, project.RepositoryPath, "diff", "--no-renames", span)
	if err != nil {
		return JobChange{}, fault(CodeInternal, "read the change", err)
	}
	if len(patch) > maxChangePatch {
		patch, change.Truncated = patch[:maxChangePatch], true
	}
	change.Patch = patch
	return change, nil
}
