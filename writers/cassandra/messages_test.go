package cassandra_test

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/writers/cassandra"
	"github.com/stretchr/testify/assert"
)

const keyspace = "mainflux"

var (
	msg  = mainflux.Message{}
	addr = "localhost"
)

func TestSave(t *testing.T) {
	session, err := cassandra.Connect([]string{addr}, keyspace)
	if err != nil {
		t.Fatalf("Failed to connect to Cassandra: %s", err)
	}

	repo := cassandra.New(session)

	err = repo.Save(msg)
	assert.Nil(t, err, fmt.Sprintf("expected no error, go %s", err))
}
