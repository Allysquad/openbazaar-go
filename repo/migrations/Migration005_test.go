package migrations

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"
)

func TestMigration005(t *testing.T) {
	os.Mkdir("./datastore", os.ModePerm)
	var m migration005
	err := m.Up("./", "letmein", false)
	if err != nil {
		t.Error(err)
	}

	_, err = os.Stat("./swarm.key")
	if err != nil {
		t.Error(err)
	}

	repoVer, err := ioutil.ReadFile("./repover")
	if err != nil {
		t.Error(err)
	}
	if string(repoVer) != "6" {
		t.Error("Failed to write new repo version")
	}

	err = m.Down("./", "letmein", false)
	if err != nil {
		t.Error(err)
		return
	}

	_, err = os.Stat("./swarm.key")
	if err == nil {
		t.Error(errors.New("Expected file to be deleted."))
	}

	repoVer, err = ioutil.ReadFile("./repover")
	if err != nil {
		t.Error(err)
	}
	if string(repoVer) != "5" {
		t.Error("Failed to write new repo version")
	}
	os.RemoveAll("./datastore")
	os.RemoveAll("./repover")
}
