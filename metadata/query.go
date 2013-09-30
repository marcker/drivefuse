// Copyright 2013 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metadata

import (
	"database/sql"
	"fmt"
	"time"
)

const (
	sqlGetByRemoteId = "select remoteId, parentId, name, mimetype, size, md5checksum, lastMod from files where remoteId = '%s'"
	sqlLookup        = "select remoteId, parentId, name, mimetype, size, md5checksum, lastMod from files where parentId = '%s' and name = '%s' and (inited = 1 or mimetype = 'application/vnd.google-apps.folder')"
	sqlChildren      = "select remoteId, parentId, name, mimetype, size, md5checksum, lastMod from files where parentId = '%s' and (inited = 1 or mimetype = 'application/vnd.google-apps.folder')"
	sqlListDownloads = "select remoteId, parentId, name, mimetype, size, md5checksum, lastMod from files where download = 1 and size >= %d and size <= %d limit %d"
	sqlUpsert        = "insert or replace into files (remoteId, parentId, name, mimetype, size, md5checksum, lastMod, download, upload) values(?, ?, ?, ?, ?, ?, ?, ?, ?)"
	sqlDelete        = "delete from files where remoteId = '%s'"
	sqlSetInited     = "update files set inited = 1 where remoteId = ?"
	sqlGetValue      = "select value from info where key = '%s'"
	sqlSetValue      = "insert or replace into info (key, value) values(?, ?)"

	layoutDateTime = "2006-01-02 15:04:05.999999999"
)

// Sets up the sqlite db, creates required tables and indexes.
func (m *MetaService) setup() error {
	queries := []string{
		"create table if not exists files (" +
			"   id integer not null primary key," +
			"   remoteId string," +
			"   parentId string," +
			"   name string," +
			"   mimetype string," +
			"   size int," +
			"   md5checksum string," +
			"   lastMod date," +
			"   inited bool default 0," +
			"   upload bool," +
			"   download bool)",
		"create table if not exists info (key string, value string)",
		"create unique index idx_remote on files (remoteId)",
		"create unique index idx_k on info (key)"}
	// don't remove the index, used by insert or replace into queries
	for _, v := range queries {
		_, err := m.db.Exec(v)
		if err != nil {
			return err
		}
	}
	return nil
}

// For the given query, returns the matching files.
func (m *MetaService) listFiles(query string) (files []*CachedDriveFile, err error) {
	var rows *sql.Rows
	if rows, err = m.db.Query(query); err != nil {
		return
	}
	defer rows.Close()

	files = []*CachedDriveFile{}
	for rows.Next() {
		var remoteId string
		var parentId string
		var name string
		var mimetype string
		var size int64
		var md5checksum string
		var lastMod string
		// TODO(burcud): add all columns
		rows.Scan(&remoteId, &parentId, &name, &mimetype, &size, &md5checksum, &lastMod)
		parsedLastMod, _ := time.Parse(layoutDateTime, lastMod)
		file := &CachedDriveFile{
			Id:          remoteId,
			ParentId:    parentId,
			Name:        name,
			MimeType:    mimetype,
			FileSize:    size,
			Md5Checksum: md5checksum,
			LastMod:     parsedLastMod,
		}
		files = append(files, file)
	}
	return
}

// Inserts/updates the given CachedDriveFile. Files are markable for
// downloading or uploading, later will be consumed by download and
// upload queues.
func (m *MetaService) upsertFile(
	file *CachedDriveFile, download bool, upload bool) (err error) {
	_, err = m.db.Exec(sqlUpsert,
		file.Id, file.ParentId, file.Name, file.MimeType, file.FileSize,
		file.Md5Checksum, file.LastMod, download, upload)
	return err
}

func (m *MetaService) updateIOQueue(name string, id string, value int) (err error) {
	_, err = m.db.Exec(fmt.Sprintf("update files set %s = %d where remoteId = '%s'", name, value, id))
	return err
}

// Deletes the file/folder identified with id.
func (m *MetaService) deleteFile(id string) error {
	_, err := m.db.Exec(fmt.Sprintf(sqlDelete, id))
	return err
}

// Gets a value.
func (m *MetaService) getValue(key string) (value string, err error) {
	var rows *sql.Rows
	if rows, err = m.db.Query(fmt.Sprintf(sqlGetValue, key)); err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&value)
		return
	}
	return
}

// Sets a value.
func (m *MetaService) setValue(key string, value string) error {
	_, err := m.db.Exec(sqlSetValue, key, value)
	return err
}
