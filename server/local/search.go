package local

import (
	"strings"

	srch "sourcegraph.com/sourcegraph/sourcegraph/search"
	"sourcegraph.com/sqs/pbtypes"

	"golang.org/x/net/context"
	"sourcegraph.com/sourcegraph/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sourcegraph/sourcegraph/store"
)

var Search sourcegraph.SearchServer = &search{}

type search struct{}

var _ sourcegraph.SearchServer = (*search)(nil)

func (s *search) Search(ctx context.Context, op *sourcegraph.SearchOp) (*sourcegraph.SearchResultsList, error) {
	var repo, unit, unitType, def string
	var descToks []string // "descriptor" tokens that don't have a special filter meaning.
	for _, token := range strings.Fields(op.Query) {
		if strings.HasPrefix(token, "r:") {
			repo = strings.TrimPrefix(token, "r:")
			continue
		}
		if strings.HasPrefix(token, "u:") {
			unit = strings.TrimPrefix(token, "u:")
			continue
		}
		if strings.HasPrefix(token, "t:") {
			unit = strings.TrimPrefix(token, "t:")
			continue
		}
		if def != "" {
			def += " "
		}
		def += token

		if strings.HasSuffix(token, ".com") || strings.HasSuffix(token, ".org") {
			descToks = append(descToks, token)
		} else {
			descToks = append(descToks, srch.QueryTokens(token)...)
		}
	}

	bowQuery := srch.UserQueryToksToTSQuery(descToks)

	results, err := store.GlobalDefsFromContext(ctx).Search(ctx, &store.GlobalDefSearchOp{
		RepoQuery:     repo,
		UnitQuery:     unit,
		DefQuery:      def, // TODO(beyang): remove this if BoW is good enough
		UnitTypeQuery: unitType,

		BoWQuery: bowQuery,
		Opt:      op.Opt,
	})
	if err != nil {
		return nil, err
	}
	for _, r := range results.Results {
		populateDefFormatStrings(&r.Def)
	}
	return results, nil
}

func (s *search) RefreshIndex(ctx context.Context, op *sourcegraph.SearchRefreshIndexOp) (*pbtypes.Void, error) {
	// Currently, the only pre-computation we do is aggregating the global ref counts
	// for every def. This will pre-compute the ref counts based on the current state
	// of the GlobalRefs table for all defs in the given repos.
	var repoURIs []string
	for _, r := range op.Repos {
		repoURIs = append(repoURIs, r.URI)
	}
	if err := store.GlobalDefsFromContext(ctx).RefreshRefCounts(ctx, repoURIs); err != nil {
		return nil, err
	}
	return &pbtypes.Void{}, nil
}
