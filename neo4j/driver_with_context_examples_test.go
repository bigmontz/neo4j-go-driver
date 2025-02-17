package neo4j

import (
	"context"
	"fmt"
)

var myDriver DriverWithContext
var ctx context.Context

func ExampleExecuteQuery() {
	query := "RETURN $value AS val"
	params := map[string]any{"value": 42}
	eagerResult, err := ExecuteQuery[*EagerResult](ctx, myDriver, query, params, EagerResultTransformer)
	handleError(err)

	// iterate over all keys (here it's only "val")
	for _, key := range eagerResult.Keys {
		fmt.Println(key)
	}
	// iterate over all records (here it's only {"val": 42})
	for _, record := range eagerResult.Records {
		rawValue, _ := record.Get("value")
		fmt.Println(rawValue.(int64))
	}
	// consume information from the query execution summary
	summary := eagerResult.Summary
	fmt.Printf("Hit database is: %s\n", summary.Database().Name())
}

func ExampleExecuteQuery_self_causal_consistency() {
	query := "CREATE (n:Example)"
	params := map[string]any{"value": 42}
	_, err := ExecuteQuery[*EagerResult](
		ctx, myDriver, query, params, EagerResultTransformer, ExecuteQueryWithWritersRouting())
	handleError(err)

	// assuming an initial empty database, the following query should return 1
	// indeed, causal consistency is guaranteed by default, which subsequent ExecuteQuery calls can read the writes of
	// previous ExecuteQuery calls targeting the same database
	query = "MATCH (n:Example) RETURN count(n) AS count"
	eagerResult, err := ExecuteQuery[*EagerResult](
		ctx, myDriver, query, nil, EagerResultTransformer, ExecuteQueryWithReadersRouting())
	handleError(err)

	// there should be a single record
	recordCount := len(eagerResult.Records)
	if recordCount != 1 {
		handleError(fmt.Errorf("expected a single record, got: %d", recordCount))
	}
	// the record should be {"count": 1}
	if rawCount, found := eagerResult.Records[0].Get("val"); !found || rawCount.(int64) != 1 {
		handleError(fmt.Errorf("expected count of 1, got: %d", rawCount.(int64)))
	}
}

func ExampleExecuteQuery_default_bookmark_manager_explicit_reuse() {
	query := "CREATE (n:Example)"
	params := map[string]any{"value": 42}
	_, err := ExecuteQuery[*EagerResult](
		ctx, myDriver, query, params, EagerResultTransformer, ExecuteQueryWithWritersRouting())
	handleError(err)

	// retrieve the default bookmark manager used by the previous call (since there was no bookmark manager explicitly
	// configured)
	bookmarkManager := myDriver.DefaultExecuteQueryBookmarkManager()
	session := myDriver.NewSession(ctx, SessionConfig{BookmarkManager: bookmarkManager})

	// the following transaction function is guaranteed to see the result of the previous query
	// since the session uses the same bookmark manager as the previous ExecuteQuery call and targets the same
	// (default) database
	count, err := session.ExecuteRead(ctx, func(tx ManagedTransaction) (any, error) {
		eagerResult, err := tx.Run(ctx, "MATCH (n:Example) RETURN count(n) AS count", nil)
		if err != nil {
			return nil, err
		}
		record, err := eagerResult.Single(ctx)
		if err != nil {
			return nil, err
		}
		count, _ := record.Get("count")
		return count.(int64), nil
	})
	handleError(err)
	fmt.Println(count)
}

func handleError(err error) {
	//lint:ignore SA9003 empty branch as the actual content depends on the reader's setup
	if err != nil {
		// do something with error
	}
}
