package dgraphdockerhelper_test

import (
	"testing"

	ddh "github.com/matthewmcneely/dgraph-docker-helper"
)

func Test_LifeCycle(t *testing.T) {
	cfg := ddh.DgraphStart(t, "")
	defer ddh.DgraphStop(t, cfg)
	ddh.DgraphLoadSchema(t, cfg, schema)

	// OK to mutate, query the graph now
}

const schema = `
type User {
    userID: ID!
    name: String!
    lastSignIn: DateTime
    recentScores: [Float]
    reputation: Int
    active: Boolean
}
`
