package codemonitors

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"hash/fnv"
	"net/http"
	"net/url"

	"github.com/graphql-go/graphql/gqlerrors"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	edb "github.com/sourcegraph/sourcegraph/enterprise/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/api/internalapi"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/featureflag"
	gitprotocol "github.com/sourcegraph/sourcegraph/internal/gitserver/protocol"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/search/client"
	"github.com/sourcegraph/sourcegraph/internal/search/commit"
	"github.com/sourcegraph/sourcegraph/internal/search/job"
	"github.com/sourcegraph/sourcegraph/internal/search/job/jobutil"
	"github.com/sourcegraph/sourcegraph/internal/search/predicate"
	"github.com/sourcegraph/sourcegraph/internal/search/repos"
	"github.com/sourcegraph/sourcegraph/internal/search/result"
	"github.com/sourcegraph/sourcegraph/internal/search/streaming"
	"github.com/sourcegraph/sourcegraph/lib/errors"
	"github.com/sourcegraph/sourcegraph/schema"
)

const gqlSettingsQuery = `query CodeMonitorSettings{
	viewerSettings {
		final
	}
}`

type gqlSettingsResponse struct {
	Data struct {
		ViewerSettings struct {
			Final string `json:"final"`
		} `json:"viewerSettings"`
	} `json:"data"`
	Errors []gqlerrors.FormattedError
}

// Settings queries for the computed Settings for the current actor
func Settings(ctx context.Context) (_ *schema.Settings, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CodeMonitorSearch")
	defer func() {
		span.LogFields(log.Error(err))
		span.Finish()
	}()

	reqBody, err := json.Marshal(map[string]interface{}{"query": gqlSettingsQuery})
	if err != nil {
		return nil, errors.Wrap(err, "marshal request body")
	}

	url, err := gqlURL("CodeMonitorSettings")
	if err != nil {
		return nil, errors.Wrap(err, "construct frontend URL")
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, errors.Wrap(err, "construct request")
	}
	req.Header.Set("Content-Type", "application/json")
	if span != nil {
		carrier := opentracing.HTTPHeadersCarrier(req.Header)
		span.Tracer().Inject(
			span.Context(),
			opentracing.HTTPHeaders,
			carrier,
		)
	}

	resp, err := httpcli.InternalDoer.Do(req.WithContext(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "do request")
	}
	defer resp.Body.Close()

	var res gqlSettingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, errors.Wrap(err, "decode response")
	}

	if len(res.Errors) > 0 {
		var combined error
		for _, err := range res.Errors {
			combined = errors.Append(combined, err)
		}
		return nil, combined
	}

	var unmarshaledSettings schema.Settings
	if err := json.Unmarshal([]byte(res.Data.ViewerSettings.Final), &unmarshaledSettings); err != nil {
		return nil, err
	}
	return &unmarshaledSettings, nil
}

func Search(ctx context.Context, db database.DB, query string, monitorID int64, settings *schema.Settings) (_ []*result.CommitMatch, err error) {
	searchClient := client.NewSearchClient(db, search.Indexed(), search.SearcherURLs())
	inputs, err := searchClient.Plan(ctx, "V2", nil, query, search.Streaming, settings, envvar.SourcegraphDotComMode())
	if err != nil {
		return nil, err
	}

	// Inline job creation so we can mutate the commit job before running it
	jobArgs := searchClient.JobArgs(inputs)
	plan, err := predicate.Expand(ctx, db, jobArgs, inputs.Plan)
	if err != nil {
		return nil, err
	}

	planJob, err := jobutil.FromExpandedPlan(jobArgs, plan, db)
	if err != nil {
		return nil, err
	}

	if featureflag.FromContext(ctx).GetBoolOr("cc-repo-aware-monitors", false) {
		hook := func(ctx context.Context, db database.DB, gs commit.GitserverClient, args *gitprotocol.SearchRequest, doSearch commit.DoSearchFunc) error {
			return hookWithID(ctx, db, gs, args, doSearch, monitorID)
		}
		planJob, err = addCodeMonitorHook(planJob, hook)
		if err != nil {
			return nil, err
		}
	}

	// Execute the search
	agg := streaming.NewAggregatingStream()
	_, err = planJob.Run(ctx, db, agg)
	if err != nil {
		return nil, err
	}

	results := make([]*result.CommitMatch, len(agg.Results))
	for i, res := range agg.Results {
		cm, ok := res.(*result.CommitMatch)
		if !ok {
			return nil, errors.Errorf("expected search to only return commit matches, but got type %T", res)
		}
		results[i] = cm
	}

	return results, nil
}

// Snapshot runs a dummy search that just saves the current state of the searched repos in the database.
// On subsequent runs, this allows us to treat all new repos or sets of args as something new that should
// be searched from the beginning.
func Snapshot(ctx context.Context, db database.DB, query string, monitorID int64, settings *schema.Settings) error {
	searchClient := client.NewSearchClient(db, search.Indexed(), search.SearcherURLs())
	inputs, err := searchClient.Plan(ctx, "V2", nil, query, search.Streaming, settings, envvar.SourcegraphDotComMode())
	if err != nil {
		return err
	}

	jobArgs := searchClient.JobArgs(inputs)
	plan, err := predicate.Expand(ctx, db, jobArgs, inputs.Plan)
	if err != nil {
		return err
	}

	planJob, err := jobutil.FromExpandedPlan(jobArgs, plan, db)
	if err != nil {
		return err
	}

	hook := func(ctx context.Context, db database.DB, gs commit.GitserverClient, args *gitprotocol.SearchRequest, _ commit.DoSearchFunc) error {
		return snapshotHook(ctx, db, gs, args, monitorID)
	}
	planJob, err = addCodeMonitorHook(planJob, hook)
	if err != nil {
		return err
	}

	_, err = planJob.Run(ctx, db, streaming.NewNullStream())
	return err
}

func addCodeMonitorHook(in job.Job, hook commit.CodeMonitorHook) (_ job.Job, err error) {
	return jobutil.MapAtom(in, func(atom job.Job) job.Job {
		switch typedAtom := atom.(type) {
		case *commit.CommitSearch:
			jobCopy := *typedAtom
			jobCopy.CodeMonitorSearchWrapper = hook
			return &jobCopy
		case *repos.ComputeExcludedRepos:
			// ComputeExcludedRepos is fine for code monitor jobs
			return atom
		default:
			err = errors.Append(err, errors.Errorf("found invalid atom job type %T for code monitor search", atom))
			return atom
		}
	}), err
}

func hookWithID(
	ctx context.Context,
	db database.DB,
	gs commit.GitserverClient,
	args *gitprotocol.SearchRequest,
	doSearch commit.DoSearchFunc,
	monitorID int64,
) error {
	cm := edb.NewEnterpriseDB(db).CodeMonitors()

	// Resolve the requested revisions into a static set of commit hashes
	commitHashes, err := gs.ResolveRevisions(ctx, args.Repo, args.Revisions)
	if err != nil {
		return err
	}

	// Look up the previously searched set of commit hashes
	argsHash := hashArgs(args)
	lastSearched, err := cm.GetLastSearched(ctx, monitorID, argsHash)
	if err != nil {
		return err
	}

	// Merge requested hashes and excluded hashes
	newRevs := make([]gitprotocol.RevisionSpecifier, 0, len(commitHashes)+len(lastSearched))
	for _, hash := range commitHashes {
		newRevs = append(newRevs, gitprotocol.RevisionSpecifier{RevSpec: hash})
	}
	for _, exclude := range lastSearched {
		newRevs = append(newRevs, gitprotocol.RevisionSpecifier{RevSpec: "^" + exclude})
	}

	// Update args with the new set of revisions
	argsCopy := *args
	argsCopy.Revisions = newRevs

	// Execute the search
	err = doSearch(&argsCopy)
	if err != nil {
		return err
	}

	// If the search was successful, store the resolved hashes
	// as the new "last searched" hashes
	return cm.UpsertLastSearched(ctx, monitorID, argsHash, commitHashes)
}

func snapshotHook(
	ctx context.Context,
	db database.DB,
	gs commit.GitserverClient,
	args *gitprotocol.SearchRequest,
	monitorID int64,
) error {
	cm := edb.NewEnterpriseDB(db).CodeMonitors()

	// Resolve the requested revisions into a static set of commit hashes
	commitHashes, err := gs.ResolveRevisions(ctx, args.Repo, args.Revisions)
	if err != nil {
		return err
	}

	argsHash := hashArgs(args)
	return cm.UpsertLastSearched(ctx, monitorID, argsHash, commitHashes)
}

func hashArgs(args *gitprotocol.SearchRequest) int64 {
	hasher := fnv.New64()
	hasher.Write([]byte(args.Repo))
	for _, rev := range args.Revisions {
		hasher.Write([]byte(rev.RevSpec))
		hasher.Write([]byte{'|'})
		hasher.Write([]byte(rev.RefGlob))
		hasher.Write([]byte{'|'})
		hasher.Write([]byte(rev.ExcludeRefGlob))
	}
	if args.Query != nil {
		hasher.Write([]byte(args.Query.String()))
	}
	binary.Write(hasher, binary.LittleEndian, args.IncludeDiff)
	return int64(hasher.Sum64())
}

func gqlURL(queryName string) (string, error) {
	u, err := url.Parse(internalapi.Client.URL)
	if err != nil {
		return "", err
	}
	u.Path = "/.internal/graphql"
	u.RawQuery = queryName
	return u.String(), nil
}
