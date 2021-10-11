package policies

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/gobwas/glob"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/gitserver"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
)

// TODO - rename, document, export
// TODO - for indexing, not retention
func IndexingExtractor(policy dbstore.ConfigurationPolicy) (maxAge *time.Duration, includeIntermediateCommits bool) {
	return policy.IndexCommitMaxAge, policy.IndexIntermediateCommits
}
func RetentionExtractor(policy dbstore.ConfigurationPolicy) (maxAge *time.Duration, includeIntermediateCommits bool) {
	return policy.RetentionDuration, policy.RetainIntermediateCommits
}

type Extractor func(policy dbstore.ConfigurationPolicy) (maxAge *time.Duration, includeIntermediateCommits bool)

type PolicyMatch struct {
	Name           string
	PolicyDuration *time.Duration
}

// TODO - document
func CommitsDescribedByPolicy(
	ctx context.Context,
	gitserverClient GitserverClient,
	repositoryID int,
	policies []dbstore.ConfigurationPolicy,
	extractor Extractor,
	includeTipOfDefaultBranch bool,
	now time.Time,
) (map[string][]PolicyMatch, error) {
	// if len(policies) == 0 {
	// TODO - required for incldueTipOfDefaultBranch
	// 	return nil, nil
	// }

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

	commitMap := map[string][]PolicyMatch{}
	branchRequests := map[string]branchRequestMeta{}

	for commit, refDescriptions := range refDescriptions {
		for _, refDescription := range refDescriptions {
			switch refDescription.Type {
			case gitserver.RefTypeBranch:
				if refDescription.IsDefaultBranch && includeTipOfDefaultBranch {
					commitMap[commit] = append(commitMap[commit], PolicyMatch{Name: refDescription.Name, PolicyDuration: nil})
				}

				forEachMatchingPolicy(policies, refDescription, dbstore.GitObjectTypeTree, patterns, extractor, now, func(policy dbstore.ConfigurationPolicy) {
					a, _ := extractor(policy) // TODO - rename
					commitMap[commit] = append(commitMap[commit], PolicyMatch{Name: refDescription.Name, PolicyDuration: a})

					// TODO - max age is dependent on upload for retention
					if maxAge, includeIntermediateCommits := extractor(policy); includeIntermediateCommits {
						branchRequests[refDescription.Name] = foldRefDescription(branchRequests[refDescription.Name], refDescription, maxAge)
					}
				})

			case gitserver.RefTypeTag:
				forEachMatchingPolicy(policies, refDescription, dbstore.GitObjectTypeTag, patterns, extractor, now, func(policy dbstore.ConfigurationPolicy) {
					a, _ := extractor(policy) // TODO - rename
					commitMap[commit] = append(commitMap[commit], PolicyMatch{Name: refDescription.Name, PolicyDuration: a})
				})
			}
		}
	}

	for branchName, x := range branchRequests {
		var maxAge *time.Duration
		// Sort in reverse order
		sort.Slice(x.maxAges, func(i, j int) bool { return x.maxAges[i] > x.maxAges[j] })
		if !x.unbounded && len(x.maxAges) > 0 {
			t := x.maxAges[0]
			maxAge = &t
		}

		// HACK HACK HACK
		var maxAge2 *time.Time
		if !includeTipOfDefaultBranch && maxAge != nil {
			t := now.Add(-*maxAge)
			maxAge2 = &t
		}

		// TODO - max age is dependent on upload for retention
		commits, err := gitserverClient.CommitsUniqueToBranch(ctx, repositoryID, branchName, x.isDefault, maxAge2)
		if err != nil {
			return nil, errors.Wrap(err, "gitserver.CommitsUniqueToBranch")
		}

		for _, commit := range commits {
			commitMap[commit] = append(commitMap[commit], PolicyMatch{Name: branchName, PolicyDuration: maxAge})
		}
	}

	for _, policy := range policies {
		if policy.Type == dbstore.GitObjectTypeCommit {
			commit, err := gitserverClient.ResolveRevision(ctx, repositoryID, policy.Pattern)
			if err != nil {
				if errcode.IsNotFound(err) {
					continue
				}

				return nil, errors.Wrap(err, "gitserver.ResolveRevision")
			}

			commitMap[string(commit)] = append(commitMap[string(commit)], PolicyMatch{Name: policy.Pattern})
		}
	}

	return commitMap, nil
}

// compilePatterns constructs a map from patterns in each given policy to a compiled glob object used
// to match to commits, branch names, and tag names. If there are multiple policies with the same pattern,
// the pattern is compiled only once.
func compilePatterns(policies []dbstore.ConfigurationPolicy) (map[string]glob.Glob, error) {
	patterns := make(map[string]glob.Glob, len(policies))

	for _, policy := range policies {
		if _, ok := patterns[policy.Pattern]; ok || policy.Type == dbstore.GitObjectTypeCommit {
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
func forEachMatchingPolicy(
	policies []dbstore.ConfigurationPolicy,
	refDescription gitserver.RefDescription,
	targetObjectType dbstore.GitObjectType,
	patterns map[string]glob.Glob,
	extractor Extractor,
	now time.Time,
	f func(policy dbstore.ConfigurationPolicy),
) {
	for _, policy := range policies {
		if policy.Type == targetObjectType && policyMatchesRefDescription(policy, refDescription, patterns, extractor, now) {
			f(policy)
		}
	}
}

// TODO - document
func policyMatchesRefDescription(
	policy dbstore.ConfigurationPolicy,
	refDescription gitserver.RefDescription,
	patterns map[string]glob.Glob,
	extractor Extractor,
	now time.Time,
) bool {
	if !patterns[policy.Pattern].Match(refDescription.Name) {
		// Name doesn't match
		return false
	}

	//
	// TODO - flag this?
	//

	// if maxAge, _ := extractor(policy); maxAge != nil { //&& now.Sub(refDescription.CreatedDate) > *maxAge {
	// fmt.Printf("OLD: %v %v > %v\n", now, now.Sub(refDescription.CreatedDate), maxAge)
	// 	// Too old
	// 	return false
	// }

	return true
}

type branchRequestMeta struct {
	isDefault bool
	unbounded bool
	maxAges   []time.Duration
}

// TODO - document
func foldRefDescription(meta branchRequestMeta, refDescription gitserver.RefDescription, maxAge *time.Duration) branchRequestMeta {
	if refDescription.IsDefaultBranch {
		meta.isDefault = true
	}

	if maxAge == nil {
		meta.unbounded = true
	} else {
		meta.maxAges = append(meta.maxAges, *maxAge)
	}

	return meta
}
