package runner

import runnerPb "github.com/computer-technology-team/go-judge/api/gen/runner"

var exitCodeToStatus = map[int]runnerPb.SubmissionStatusUpdate_Status{
	137: runnerPb.SubmissionStatusUpdate_MEMORY_LIMIT_EXCEEDED,
	124: runnerPb.SubmissionStatusUpdate_TIME_LIMIT_EXCEEDED,
	2:   runnerPb.SubmissionStatusUpdate_WRONG_ANSWER,
	3:   runnerPb.SubmissionStatusUpdate_INTERNAL_ERROR,
	127: runnerPb.SubmissionStatusUpdate_INTERNAL_ERROR,
	0:   runnerPb.SubmissionStatusUpdate_ACCEPTED,
}
