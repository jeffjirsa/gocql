// +build all integration

package gocql

import (
	"fmt"
	"testing"
)

type position struct {
	Lat int
	Lon int
}

// NOTE: due to current implementation details it is not currently possible to use
// a pointer receiver type for the UDTMarshaler interface to handle UDT's
func (p position) MarshalUDT(name string, info TypeInfo) ([]byte, error) {
	switch name {
	case "lat":
		return Marshal(info, p.Lat)
	case "lon":
		return Marshal(info, p.Lon)
	default:
		return nil, fmt.Errorf("unknown column for position: %q", name)
	}
}

func (p *position) UnmarshalUDT(name string, info TypeInfo, data []byte) error {
	switch name {
	case "lat":
		return Unmarshal(info, data, &p.Lat)
	case "lon":
		return Unmarshal(info, data, &p.Lon)
	default:
		return fmt.Errorf("unknown column for position: %q", name)
	}
}

func TestUDT_Marshaler(t *testing.T) {
	if *flagProto < protoVersion3 {
		t.Skip("UDT are only available on protocol >= 3")
	}

	session := createSession(t)
	defer session.Close()

	err := createTable(session, `CREATE TYPE position(
		lat int,
		lon int);`)
	if err != nil {
		t.Fatal(err)
	}

	err = createTable(session, `CREATE TABLE houses(
		id int,
		name text,
		loc frozen<position>,

		primary key(id)
	);`)
	if err != nil {
		t.Fatal(err)
	}

	const (
		expLat = -1
		expLon = 2
	)

	err = session.Query("INSERT INTO houses(id, name, loc) VALUES(?, ?, ?)", 1, "test", &position{expLat, expLon}).Exec()
	if err != nil {
		t.Fatal(err)
	}

	pos := &position{}

	err = session.Query("SELECT loc FROM houses WHERE id = ?", 1).Scan(pos)
	if err != nil {
		t.Fatal(err)
	}

	if pos.Lat != expLat {
		t.Errorf("expeceted lat to be be %d got %d", expLat, pos.Lat)
	}
	if pos.Lon != expLon {
		t.Errorf("expeceted lon to be be %d got %d", expLon, pos.Lon)
	}
}

func TestUDT_Reflect(t *testing.T) {
	if *flagProto < protoVersion3 {
		t.Skip("UDT are only available on protocol >= 3")
	}

	// Uses reflection instead of implementing the marshaling type
	session := createSession(t)
	defer session.Close()

	err := createTable(session, `CREATE TYPE horse(
		name text,
		owner text);`)
	if err != nil {
		t.Fatal(err)
	}

	err = createTable(session, `CREATE TABLE horse_race(
		position int,
		horse frozen<horse>,

		primary key(position)
	);`)
	if err != nil {
		t.Fatal(err)
	}

	type horse struct {
		Name  string `cql:"name"`
		Owner string `cql:"owner"`
	}

	insertedHorse := &horse{
		Name:  "pony",
		Owner: "jim",
	}

	err = session.Query("INSERT INTO horse_race(position, horse) VALUES(?, ?)", 1, insertedHorse).Exec()
	if err != nil {
		t.Fatal(err)
	}

	retrievedHorse := &horse{}
	err = session.Query("SELECT horse FROM horse_race WHERE position = ?", 1).Scan(retrievedHorse)
	if err != nil {
		t.Fatal(err)
	}

	if *retrievedHorse != *insertedHorse {
		t.Fatal("exepcted to get %+v got %+v", insertedHorse, retrievedHorse)
	}
}

func TestUDT_Proto2error(t *testing.T) {
	if *flagProto < protoVersion3 {
		t.Skip("UDT are only available on protocol >= 3")
	}

	cluster := createCluster()
	cluster.ProtoVersion = 2
	cluster.Keyspace = "gocql_test"

	// Uses reflection instead of implementing the marshaling type
	session, err := cluster.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	err = createTable(session, `CREATE TYPE fish(
		name text,
		owner text);`)
	if err != nil {
		t.Fatal(err)
	}

	err = createTable(session, `CREATE TABLE fish_race(
		position int,
		fish frozen<fish>,

		primary key(position)
	);`)
	if err != nil {
		t.Fatal(err)
	}

	type fish struct {
		Name  string `cql:"name"`
		Owner string `cql:"owner"`
	}

	insertedFish := &fish{
		Name:  "pony",
		Owner: "jim",
	}

	err = session.Query("INSERT INTO fish_race(position, fish) VALUES(?, ?)", 1, insertedFish).Exec()
	if err != ErrorUDTUnavailable {
		t.Fatalf("expected to get %v got %v", ErrorUDTUnavailable, err)
	}
}
