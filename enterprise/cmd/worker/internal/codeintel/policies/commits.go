package policies

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/gobwas/glob"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/gitserver"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
)

// TODO  - rename, document, export
// TODO - for indexing, not retention
func extractor(policy dbstore.ConfigurationPolicy) (maxAge *time.Duration, includeIntermediateCommits bool) {
	return policy.IndexCommitMaxAge, policy.IndexIntermediateCommits
}

// TODO - rename
// TODO - document
// TODO - make private
type SR struct {
	isDefault bool
	unbounded bool
	maxAge    *time.Duration
}

// TODO - document
// TODO - rewrite upload expirer to use this
func CommitsDescribedByPolicy(ctx context.Context, gitserverClient GitserverClient, repositoryID int, policies []dbstore.ConfigurationPolicy) ([]string, error) {
	if len(policies) == 0 {
		return nil, nil
	}

	// Get a list of relevant branch and tag heads of this repository
	refDescriptions, err := gitserverClient.RefDescriptions(ctx, repositoryID)
	if err != nil {
		return nil, errors.Wrap(err, "gitserver.RefDescriptions")
	}

	// Pre-compile the glob patterns in all the policies to reduce the number of times we need to compile
	// the pattern in the loops below.
	patterns, err := compilePatterns(policies)
	if err != nil {
		return nil, err
	}

	commitMap := map[string]struct{}{}
	branchRequests := map[string]SR{}

	for commit, refDescriptions := range refDescriptions {
		for _, refDescription := range refDescriptions {
			switch refDescription.Type {
			case gitserver.RefTypeBranch:
				forEachMatchingPolicy(policies, refDescription, dbstore.GitObjectTypeTag, patterns, func(policy dbstore.ConfigurationPolicy) {
					commitMap[commit] = struct{}{}

					maxAge, includeIntermediateCommits := extractor(policy)
					if !includeIntermediateCommits {
						return
					}

					branchRequests[refDescription.Name] = foldRefDescription(branchRequests[refDescription.Name], refDescription, maxAge)
				})

			case gitserver.RefTypeTag:
				forEachMatchingPolicy(policies, refDescription, dbstore.GitObjectTypeTag, patterns, func(policy dbstore.ConfigurationPolicy) {
					commitMap[commit] = struct{}{}
				})
			}
		}
	}

	for branchName, x := range branchRequests {
		var maxAge *time.Time
		if x.maxAge != nil {
			t := time.Now().Add(-*x.maxAge)
			maxAge = &t
		}

		commits, err := gitserverClient.CommitsUniqueToBranch(ctx, repositoryID, branchName, x.isDefault, maxAge)
		if err != nil {
			return nil, errors.Wrap(err, "gitserver.CommitsUniqueToBranch")
		}

		for _, commit := range commits {
			commitMap[commit] = struct{}{}
		}
	}

	for _, policy := range policies {
		if policy.Type == dbstore.GitObjectTypeCommit {
			// TODO - not sure how to incorporate this
			log15.Warn(fmt.Sprintf("WTF DO I DO WITH: %v\n", policy))
		}
	}

	commits := make([]string, 0, len(commitMap))
	for commit := range commitMap {
		commits = append(commits, commit)
	}
	sort.Strings(commits)

	return commits, nil
}

// compilePatterns constructs a map from patterns in each given policy to a compiled glob object used
// to match to commits, branch names, and tag names. If there are multiple policies with the same pattern,
// the pattern is compiled only once.
func compilePatterns(policies []dbstore.ConfigurationPolicy) (map[string]glob.Glob, error) {
	patterns := make(map[string]glob.Glob, len(policies))

	for _, policy := range policies {
		if _, ok := patterns[policy.Pattern]; ok {
			continue
		}

		pattern, err := glob.Compile(policy.Pattern)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to compile glob pattern `%s` in configuration policy %d", policy.Pattern, policy.ID))
		}

		patterns[policy.Pattern] = pattern
	}

	return patterns, nil
}

// TODO - document
func forEachMatchingPolicy(policies []dbstore.ConfigurationPolicy, refDescription gitserver.RefDescription, targetObjectType dbstore.GitObjectType, patterns map[string]glob.Glob, f func(policy dbstore.ConfigurationPolicy)) {
	for _, policy := range policies {
		if policy.Type == targetObjectType && policyMatchesRefDescription(policy, refDescription, patterns) {
			f(policy)
		}
	}
}

// TODO - document
func policyMatchesRefDescription(policy dbstore.ConfigurationPolicy, refDescription gitserver.RefDescription, patterns map[string]glob.Glob) bool {
	if !patterns[policy.Pattern].Match(refDescription.Name) {
		// Name doesn't match
		return false
	}

	if maxAge, _ := extractor(policy); maxAge != nil && time.Since(refDescription.CreatedDate) > *maxAge {
		// Too old
		return false
	}

	return true
}

// TODO - document
func foldRefDescription(sr SR, refDescription gitserver.RefDescription, maxAge *time.Duration) SR {
	if refDescription.IsDefaultBranch {
		sr.isDefault = true
	}

	if maxAge == nil {
		sr.unbounded = true
	} else if sr.maxAge == nil || *maxAge < *sr.maxAge {
		sr.maxAge = maxAge
	}

	return sr
}
