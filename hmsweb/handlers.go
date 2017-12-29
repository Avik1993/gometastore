// Copyright © 2017 Alex Kolbasov
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"math/rand"
	"time"

	"github.com/akolb1/gometastore/hmsclient"
	"github.com/akolb1/gometastore/hmsclient/thrift/gen-go/hive_metastore"
	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
	"github.com/oklog/ulid"
)

// getClient connects to the host specified in the requests and returns connected HMS client.
func getClient(w http.ResponseWriter, r *http.Request) (*hmsclient.MetastoreClient, error) {
	vars := mux.Vars(r)
	server := vars[paramHost]
	if server == "" {
		server = "localhost"
	}
	client, err := hmsclient.Open(server, hmsPort)
	if err != nil {
		w.Header().Set("X-HMS-Error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return nil, err
	}
	return client, err
}

// getULID returns a unique ID.
func getULID() string {
	t := time.Unix(1000000, 0)
	entropy := rand.New(rand.NewSource(t.UnixNano()))
	return ulid.MustNew(ulid.Timestamp(t), entropy).String()
}

// showError shows error information in X-HMS-Error header
func showError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("X-HMS-Error", err.Error())
	w.WriteHeader(code)
}

// showHelp shows a link to the documentation. It is served on '/' route.
func showHelp(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1>%s</h1><div>"+
		"See <a href=%s>Documentation</a></div>",
		"HmsWEB - HTTP interface to Hive Metastore",
		"https://github.com/akolb1/gometastore/tree/master/hmsweb")
}

// databaseList shows list of databases.
func databaseList(w http.ResponseWriter, r *http.Request) {
	client, err := getClient(w, r)
	if err != nil {
		return
	}
	defer client.Close()
	databases, err := client.GetAllDatabases()
	if err != nil {
		showError(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", jsonEncoding)

	// Either show full URI for each database or show compact presentation -
	// just list of databases, based on "Compact" query parameter
	compact, _ := strconv.ParseBool(r.URL.Query().Get("Compact"))
	if !compact {
		dbList := make([]string, len(databases))
		for i, d := range databases {
			dbList[i] = r.Host + r.URL.Path + "/" + d
		}
		databases = dbList
	}
	json.NewEncoder(w).Encode(databases)
}

// databaseShow displays information about the database.
func databaseShow(w http.ResponseWriter, r *http.Request) {
	client, err := getClient(w, r)
	if err != nil {
		return
	}
	defer client.Close()
	vars := mux.Vars(r)
	dbName := vars[paramDbName]
	database, err := client.GetDatabase(dbName)
	if err != nil {
		showError(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", jsonEncoding)
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(database)
}

// databaseCreate creates a new database
func databaseCreate(w http.ResponseWriter, r *http.Request) {
	client, err := getClient(w, r)
	if err != nil {
		return
	}
	defer client.Close()
	vars := mux.Vars(r)
	var db hmsclient.Database
	_ = json.NewDecoder(r.Body).Decode(&db)
	db.Name = vars[paramDbName]
	if db.Owner == "" {
		db.Owner = r.URL.Query().Get("owner")
	}

	// Add ULID to parameters
	if db.Parameters == nil {
		db.Parameters = make(map[string]string)
	}
	if db.Parameters["ULID"] == "" {
		db.Parameters["ULID"] = getULID()
	}

	log.Println(fmt.Sprintf("Creating database %#v", db))
	err = client.CreateDatabase(&db)
	if err != nil {
		showError(w, http.StatusBadRequest, err)
		return
	}
	database, err := client.GetDatabase(db.Name)
	if err != nil {
		showError(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", jsonEncoding)
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(database)
}

func databaseDrop(w http.ResponseWriter, r *http.Request) {
	client, err := getClient(w, r)
	if err != nil {
		return
	}
	defer client.Close()
	vars := mux.Vars(r)
	dbName := vars[paramDbName]
	deleteData, _ := strconv.ParseBool(r.URL.Query().Get("data"))
	cascade, _ := strconv.ParseBool(r.URL.Query().Get("cascade"))
	log.Println("Drop database", dbName, "d =", deleteData, "c =", cascade)
	err = client.DropDatabase(dbName, deleteData, cascade)
	if err != nil {
		showError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func tablesList(w http.ResponseWriter, r *http.Request) {
	client, err := getClient(w, r)
	if err != nil {
		return
	}
	defer client.Close()
	vars := mux.Vars(r)
	dbName := vars[paramDbName]
	tables, err := client.GetAllTables(dbName)
	if err != nil {
		showError(w, http.StatusBadRequest, err)
		return
	}
	// Either show full URI for each database or show compact presentation -
	// just list of databases, based on "Compact" query parameter
	compact, _ := strconv.ParseBool(r.URL.Query().Get("Compact"))
	if !compact {
		tblList := make([]string, len(tables))
		for i, t := range tables {
			tblList[i] = r.Host + r.URL.Path + t
		}
		tables = tblList
	}

	w.Header().Set("Content-Type", jsonEncoding)
	json.NewEncoder(w).Encode(tables)
}

func tablesShow(w http.ResponseWriter, r *http.Request) {
	client, err := getClient(w, r)
	if err != nil {
		return
	}
	defer client.Close()
	vars := mux.Vars(r)
	dbName := vars[paramDbName]
	tableName := vars[paramTblName]
	table, err := client.GetTable(dbName, tableName)
	if err != nil {
		showError(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", jsonEncoding)

	json.NewEncoder(w).Encode(table)
}

func tableCreate(w http.ResponseWriter, r *http.Request) {
	client, err := getClient(w, r)
	if err != nil {
		return
	}
	defer client.Close()
	vars := mux.Vars(r)

	type Table struct {
		Columns    []hive_metastore.FieldSchema `json:"columns"`
		Partitions []hive_metastore.FieldSchema `json:"partitions"`
		Owner      string                       `json:"owner"`
		Parameters map[string]string            `json:"parameters"`
	}

	dbName := vars[paramDbName]
	tableName := vars[paramTblName]
	var tbl Table
	_ = json.NewDecoder(r.Body).Decode(&tbl)
	if tbl.Owner == "" {
		tbl.Owner = r.URL.Query().Get("owner")
	}
	// Add ULID to parameters
	if tbl.Parameters == nil {
		tbl.Parameters = make(map[string]string)
	}
	if tbl.Parameters["ULID"] == "" {
		tbl.Parameters["ULID"] = getULID()
	}

	table := hmsclient.MakeTable(dbName, tableName, tbl.Owner, tbl.Parameters, tbl.Columns, tbl.Partitions)
	log.Println("Creating table " + spew.Sdump(table))
	err = client.CreateTable(table)
	if err != nil {
		showError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func tableDrop(w http.ResponseWriter, r *http.Request) {
	client, err := getClient(w, r)
	if err != nil {
		return
	}
	defer client.Close()
	vars := mux.Vars(r)
	dbName := vars[paramDbName]
	tableName := vars[paramTblName]
	deleteData, _ := strconv.ParseBool(r.URL.Query().Get("data"))
	log.Println("Drop table", dbName, tableName, "d =", deleteData)
	err = client.DropTable(dbName, tableName, deleteData)
	if err != nil {
		showError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func partitionsList(w http.ResponseWriter, r *http.Request) {
	client, err := getClient(w, r)
	if err != nil {
		return
	}
	defer client.Close()
	vars := mux.Vars(r)
	dbName := vars[paramDbName]
	tableName := vars[paramTblName]
	partitions, err := client.GetPartitionNames(dbName, tableName, -1)
	if err != nil {
		showError(w, http.StatusBadRequest, err)
		return
	}
	// Either show full URI for each database or show compact presentation -
	// just list of databases, based on "Compact" query parameter
	compact, _ := strconv.ParseBool(r.URL.Query().Get("Compact"))
	if !compact {
		pList := make([]string, len(partitions))
		for i, t := range partitions {
			url := r.URL
			pList[i] = r.Host + url.Path + t
		}
		partitions = pList
	}

	w.Header().Set("Content-Type", jsonEncoding)
	json.NewEncoder(w).Encode(partitions)
}

func partitionShow(w http.ResponseWriter, r *http.Request) {
	client, err := getClient(w, r)
	if err != nil {
		return
	}
	defer client.Close()
	vars := mux.Vars(r)
	dbName := vars[paramDbName]
	tableName := vars[paramTblName]
	partName := vars[paramPartName]
	partition, err := client.GetPartitionByName(dbName, tableName, partName)
	if err != nil {
		showError(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", jsonEncoding)
	json.NewEncoder(w).Encode(partition)
}

func partitionAdd(w http.ResponseWriter, r *http.Request) {
	client, err := getClient(w, r)
	if err != nil {
		return
	}
	defer client.Close()
	vars := mux.Vars(r)
	dbName := vars[paramDbName]
	tableName := vars[paramTblName]

	type Partition struct {
		Values     []string          `json:"values"`
		Parameters map[string]string `json:"parameters"`
	}
	var part Partition
	_ = json.NewDecoder(r.Body).Decode(&part)
	table, err := client.GetTable(dbName, tableName)
	if err != nil {
		showError(w, http.StatusBadRequest, err)
		return
	}
	partition, err := hmsclient.MakePartition(table, part.Values, part.Parameters)
	log.Println("Creating partition " + spew.Sdump(partition))
	if err != nil {
		showError(w, http.StatusBadRequest, err)
		return
	}
	newPart, err := client.AddPartition(partition)
	if err != nil {
		showError(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", jsonEncoding)
	json.NewEncoder(w).Encode(newPart)
}
