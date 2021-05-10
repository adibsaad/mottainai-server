/*

Copyright (C) 2021  Daniele Rondina <geaaru@sabayonlinux.org>
Credits goes also to Gogs authors, some code portions and re-implemented design
are also coming from the Gogs project, which is using the go-macaron framework
and was really source of ispiration. Kudos to them!

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

package tiedot

import (
	"strconv"

	"github.com/MottainaiCI/mottainai-server/pkg/queues"

	dbcommon "github.com/MottainaiCI/mottainai-server/pkg/db/common"
)

var QueueColl = "Queues"

func (d *Database) IndexQueue() {
	d.AddIndex(QueueColl, []string{"qid"})
	d.AddIndex(QueueColl, []string{"name"})
}

func (d *Database) CreateQueue(t map[string]interface{}) (string, error) {
	return d.InsertDoc(QueueColl, t)
}

func (d *Database) InsertQueue(q *queues.Queue) (string, error) {
	return d.CreateQueue(q.ToMap())
}

func (d *Database) DeleteQueue(docId string) error {
	return d.DeleteDoc(QueueColl, docId)
}

func (d *Database) UpdateQueue(docId string, t map[string]interface{}) error {
	return d.UpdateDoc(QueueColl, docId, t)
}

func (d *Database) GetQueue(docId string) (queues.Queue, error) {
	doc, err := d.GetDoc(QueueColl, docId)
	if err != nil {
		return queues.Queue{}, err
	}

	t := queues.NewQueueFromMap(doc)
	t.ID = docId
	return t, err
}

func (d *Database) GetQueueByKey(name string) (queues.Queue, error) {
	var res []queues.Queue

	queuesFound, err := d.FindDoc(QueueColl, `[{"eq": "`+name+`", "in": ["name"]}]`)
	if err != nil || len(queuesFound) != 1 {
		return queues.Queue{}, nil
	}

	for docid := range queuesFound {
		q, err := d.GetQueue(docid)
		q.ID = docid
		if err != nil {
			return queues.Queue{}, err
		}
		res = append(res, q)
	}

	return res[0], nil
}

func (d *Database) ListQueues() []dbcommon.DocItem {
	return d.ListDocs(QueueColl)
}

func (d *Database) AllQueues() []queues.Queue {
	queuec := d.DB().Use(QueueColl)
	queue_list := make([]queues.Queue, 0)

	queuec.ForEachDoc(func(id int, docContent []byte) (willMoveOn bool) {
		t := queues.NewFromJson(docContent)
		t.ID = strconv.Itoa(id)
		queue_list = append(queue_list, t)
		return true
	})
	return queue_list
}