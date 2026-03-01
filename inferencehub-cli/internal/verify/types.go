package verify

import "time"

// VerificationResult represents the result of a verification step
type VerificationResult struct {
	Step     string
	Success  bool
	Message  string
	Duration time.Duration
	Details  string
}

// VerificationSummary holds the summary of all verification results
type VerificationSummary struct {
	Results      []VerificationResult
	TotalSteps   int
	SuccessCount int
	FailureCount int
	TotalTime    time.Duration
}

// AllPassed returns true if all verification steps passed
func (s *VerificationSummary) AllPassed() bool {
	return s.FailureCount == 0 && s.SuccessCount == s.TotalSteps
}

// Add adds a result to the summary
func (s *VerificationSummary) Add(result VerificationResult) {
	s.Results = append(s.Results, result)
	s.TotalSteps++
	s.TotalTime += result.Duration

	if result.Success {
		s.SuccessCount++
	} else {
		s.FailureCount++
	}
}
