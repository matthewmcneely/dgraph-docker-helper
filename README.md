## dgraph-docker-helper

A butt-simple test helper to use a transient Dgraph cluster in tests.

Example
```golang
import (
	"testing"

	ddh "github.com/matthewmcneely/dgraph-docker-helper"
)

func Test_LifeCycle(t *testing.T) {
	cfg := ddh.DgraphStart(t, "")
	defer ddh.DgraphStop(t, cfg)
	ddh.DgraphLoadSchema(t, cfg, schema)

	// OK to mutate/query the graph now
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
```

#### Requirements

* Docker
* The Dgraph image in your Docker cache (e.g., `docker pull dgraph/standalone:v21.03.1`)