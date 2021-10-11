package janitor

import (
	"context"
	"time"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/gitserver"
	api "github.com/sourcegraph/sourcegraph/internal/api"
)

// testUploadExpirerMockGitserverClient returns a mock GitserverClient instance that
// has default behaviors useful for testing the upload expirer.
func testUploadExpirerMockGitserverClient(branchMap map[string]map[string]string, tagMap map[string][]string) *MockGitserverClient {
	gitserverClient := NewMockGitserverClient()

	gitserverClient.ResolveRevisionFunc.SetDefaultHook(func(ctx context.Context, repositoryID int, revlike string) (api.CommitID, error) {
		return api.CommitID(revlike), nil
	})

	gitserverClient.RefDescriptionsFunc.SetDefaultHook(func(ctx context.Context, repositoryID int) (map[string][]gitserver.RefDescription, error) {
		refDescriptions := map[string][]gitserver.RefDescription{}
		for commit, branches := range branchMap {
			for branch, tip := range branches {
				if tip != commit {
					continue
				}

				refDescriptions[commit] = append(refDescriptions[commit], gitserver.RefDescription{
					Name:            branch,
					Type:            gitserver.RefTypeBranch,
					IsDefaultBranch: branch == "main",
				})
			}
		}

		for commit, tags := range tagMap {
			for _, tag := range tags {
				refDescriptions[commit] = append(refDescriptions[commit], gitserver.RefDescription{
					Name: tag,
					Type: gitserver.RefTypeTag,
				})
			}
		}

		return refDescriptions, nil
	})

	gitserverClient.CommitsUniqueToBranchFunc.SetDefaultHook(func(ctx context.Context, repositoryID int, branchName string, isDefaultBranch bool, maxAge *time.Time) (branches []string, _ error) {
		for commit, branchMap := range branchMap {
			if _, ok := branchMap[branchName]; ok {
				branches = append(branches, commit)
			}
		}

		return branches, nil
	})

	return gitserverClient
}
