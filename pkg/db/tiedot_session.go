/*

Copyright (C) 2018  Ettore Di Giacinto <mudler@gentoo.org>
Copyright (C) 2020  Adib Saad <adib.saad@gmail.com>
Credits goes also to Gogs authors, some code portions and re-implemented design
are also coming from the Gogs project, which is using the go-macaron framework
and was really source of inspiration. Kudos to them!

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.

*/

package database

import (
  "encoding/base64"
  "errors"
  "fmt"
  "github.com/HouzuoGuo/tiedot/db"
  "github.com/HouzuoGuo/tiedot/dberr"
  "github.com/MottainaiCI/mottainai-server/pkg/db/tiedot"
  "github.com/go-macaron/session"
  "sync"
  "time"
)

type TiedotStore struct {
  d     *tiedot.Database
  sid   string
  lock  sync.RWMutex
  data  map[interface{}]interface{}
}

// NewRedisStore creates and returns a redis session store.
func NewTiedotStore(d *tiedot.Database, sid string, kv map[interface{}]interface{}) *TiedotStore {
  return &TiedotStore{
    d:     d,
    sid:   sid,
    data:  kv,
  }
}

// Set sets value to given key in session.
func (s *TiedotStore) Set(key, val interface{}) error {
  s.lock.Lock()
  defer s.lock.Unlock()

  s.data[key] = val
  return nil
}

// Get gets value by given key in session.
func (s *TiedotStore) Get(key interface{}) interface{} {
  s.lock.RLock()
  defer s.lock.RUnlock()

  return s.data[key]
}

// Delete delete a key from session.
func (s *TiedotStore) Delete(key interface{}) error {
  s.lock.Lock()
  defer s.lock.Unlock()

  delete(s.data, key)
  return nil
}

// ID returns current session ID.
func (s *TiedotStore) ID() string {
  return s.sid
}

// Release releases resource and save data to provider.
func (s *TiedotStore) Release() error {
  // Skip encoding if the data is empty
  if len(s.data) == 0 {
    return nil
  }

  data, err := session.EncodeGob(s.data)
  if err != nil {
    return err
  }

  sessions := s.d.DB().Use(tiedot.SessionColl)

  query := map[string]interface{}{
    "eq": s.sid,
    "in": []interface{}{"key"},
  }

  queryResult := make(map[int]struct{})
  if err := db.EvalQuery(query, sessions, &queryResult); err != nil {
    return err
  }

  var docId int
  for id := range queryResult {
    docId = id
    break
  }
  storedData := base64.StdEncoding.EncodeToString(data)
  err = sessions.Update(docId, map[string]interface{}{
    "key":    s.sid,
    "data":   storedData,
    "expiry": time.Now().Unix()})
  return err
}

// Flush deletes all session data.
func (s *TiedotStore) Flush() error {
  s.lock.Lock()
  defer s.lock.Unlock()

  s.data = make(map[interface{}]interface{})
  return nil
}

type TiedotProvider struct {
  d      *tiedot.Database
  expire int64
}

// Init initializes mysql session provider.
func (p *TiedotProvider) Init(expire int64, connStr string) (err error) {
  p.expire = expire

  switch f := DBInstance.Driver.(type) {
  case *tiedot.Database:
    p.d = f
  default:
    return errors.New("unsupported driver")
  }
  return nil
}

func (p *TiedotProvider) Read(sid string) (session.RawStore, error) {
  var data []byte
  var err error
  query := map[string]interface{}{
    "eq": sid,
    "in": []interface{}{"key"},
  }

  queryResult := make(map[int]struct{})
  sessions := p.d.DB().Use(tiedot.SessionColl)
  if err := db.EvalQuery(query, sessions, &queryResult); err != nil {
    return nil, err
  }

  if len(queryResult) == 0 {
    _, err := sessions.Insert(map[string]interface{}{
      "key":    sid,
      "data":   "",
      "expiry": time.Now().Unix()})
    if err != nil {
      return nil, err
    }
    return NewTiedotStore(p.d, sid, make(map[interface{}]interface{})), nil
  }

  storedData := ""
  for id := range queryResult {
    // To get query result document, simply read it
    readBack, err := sessions.Read(id)
    if err != nil {
      return nil, err
    }

    storedData = readBack["data"].(string)
  }

  data, _ = base64.StdEncoding.DecodeString(storedData)
  var kv map[interface{}]interface{}
  if len(data) == 0 {
    kv = make(map[interface{}]interface{})
  } else {
    kv, err = session.DecodeGob(data)
    if err != nil {
      return nil, err
    }
  }

  return NewTiedotStore(p.d, sid, kv), nil
}

// Exist returns true if session with given ID exists.
func (p *TiedotProvider) Exist(sid string) bool {
  query := map[string]interface{}{
    "eq": sid,
    "in": []interface{}{"key"},
  }

  queryResult := make(map[int]struct{})
  sessions := p.d.DB().Use(tiedot.SessionColl)
  if err := db.EvalQuery(query, sessions, &queryResult); err != nil {
    return false
  }

  return len(queryResult) > 0
}

// Destory deletes a session by session ID.
func (p *TiedotProvider) Destory(sid string) error {
  query := map[string]interface{}{
    "eq": sid,
    "in": []interface{}{"key"},
  }

  queryResult := make(map[int]struct{})
  sessions := p.d.DB().Use(tiedot.SessionColl)
  if err := db.EvalQuery(query, sessions, &queryResult); err != nil {
    return err
  }

  for id := range queryResult {
    if err := sessions.Delete(id); dberr.Type(err) == dberr.ErrorNoDoc {
      fmt.Println("The document was already deleted")
    }
  }

  return nil
}

// Regenerate regenerates a session store from old session ID to new one.
func (p *TiedotProvider) Regenerate(oldsid, sid string) (_ session.RawStore, err error) {
  if p.Exist(sid) {
    return nil, fmt.Errorf("new sid '%s' already exists", sid)
  }

  if !p.Exist(oldsid) {
    sessions := p.d.DB().Use(tiedot.SessionColl)
    _, err := sessions.Insert(map[string]interface{}{
      "key":    oldsid,
      "data":   "",
      "expiry": time.Now().Unix()})
    if err != nil {
      return nil, err
    }
  }

  query := map[string]interface{}{
    "eq": oldsid,
    "in": []interface{}{"key"},
  }

  queryResult := make(map[int]struct{})
  sessions := p.d.DB().Use(tiedot.SessionColl)
  if err := db.EvalQuery(query, sessions, &queryResult); err != nil {
    return nil, err
  }

  var docId int
  for id := range queryResult {
    docId = id
    break
  }

  if err := sessions.Update(docId, map[string]interface{}{"key": sid}); err != nil {
    return nil, err
  }

  return p.Read(sid)
}

// Count counts and returns number of sessions.
func (p *TiedotProvider) Count() (total int) {
  queryResult := make(map[int]struct{})
  sessions := p.d.DB().Use(tiedot.SessionColl)
  if err := db.EvalQuery("all", sessions, &queryResult); err != nil {
    return 0
  }

  return len(queryResult)
}

// GC calls GC to clean expired sessions.
func (p *TiedotProvider) GC() {
  query := map[string]interface{}{
    "int-to": time.Now().Unix(),
    "in": []interface{}{"expiry"},
  }

  queryResult := make(map[int]struct{})
  sessions := p.d.DB().Use(tiedot.SessionColl)
  if err := db.EvalQuery(query, sessions, &queryResult); err != nil {
    return
  }

  for id := range queryResult {
    if err := sessions.Delete(id); dberr.Type(err) == dberr.ErrorNoDoc {
      fmt.Println("The document was already deleted")
    }
  }
}

func init() {
  session.Register("tiedot", &TiedotProvider{})
}
