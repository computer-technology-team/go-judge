package storage

import runnerPb "github.com/computer-technology-team/go-judge/api/gen/runner"

func (tc *TestCase) ToProto() *runnerPb.SubmissionRequest_TestCase {
	return &runnerPb.SubmissionRequest_TestCase{
		Input:  tc.Input,
		Output: tc.Output,
	}
}
