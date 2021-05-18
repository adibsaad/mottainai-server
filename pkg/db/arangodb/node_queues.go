/*

Copyright (C) 2021 Daniele Rondina <geaaru@sabayonlinux.org>
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

package arangodb

import (
	"errors"

	"github.com/MottainaiCI/mottainai-server/pkg/queues"

	dbcommon "github.com/MottainaiCI/mottainai-server/pkg/db/common"
)

var NodeQueuesColl = "NodeQueues"

func (d *Database) IndexNodeQueue() {
	d.AddIndex(NodeQueuesColl, []string{"akey"})
}

func (d *Database) CreateNodeQueues(t map[string]interface{}) (string, error) {
	return d.InsertDoc(NodeQueuesColl, t)
}

func (d *Database) InsertNodeQueues(q *queues.NodeQueues) (string, error) {
	return d.CreateNodeQueues(q.ToMap())
}

func (d *Database) DeleteNodeQueues(docId string) error {
	return d.DeleteDoc(NodeQueuesColl, docId)
}

func (d *Database) UpdateNodeQueues(docId string, t map[string]interface{}) error {
	return d.UpdateDoc(NodeQueuesColl, docId, t)
}

func (d *Database) GetNodeQueues(docId string) (queues.NodeQueues, error) {
	return queues.NodeQueues{}, errors.New("Not implemented")
}

func (d *Database) GetNodeQueuesByKey(agentKey, nodeid string) (queues.NodeQueues, error) {
	return queues.NodeQueues{}, errors.New("Not implemented")
}

func (d *Database) AddNodeQueuesTask(agentKey, nodeid, queue, taskid string) error {
	// TODO: add a semaphore
	nq, err := d.GetNodeQueuesByKey(agentKey, nodeid)
	if err != nil {
		return err
	}

	if _, ok := nq.Queues[queue]; !ok {
		nq.Queues[queue] = []string{}
	}

	nq.Queues[queue] = append(nq.Queues[queue], taskid)

	err = d.UpdateNodeQueues(nq.ID, map[string]interface{}{
		"akey":          nq.AgentKey,
		"nodeid":        nq.NodeId,
		"queues":        nq.Queues,
		"creation_date": nq.CreationDate,
	})

	return err
}

func (d *Database) DelNodeQueuesTask(agentKey, nodeid, queue, taskid string) error {
	// TODO: Add a semaphore

	nq, err := d.GetNodeQueuesByKey(agentKey, nodeid)
	if err != nil {
		return err
	}

	if _, ok := nq.Queues[queue]; ok {
		tasks := nq.Queues[queue]

		ntasks := []string{}

		for _, t := range tasks {
			if t == taskid {
				continue
			}
			ntasks = append(ntasks, t)
		}

		nq.Queues[queue] = ntasks
	}

	err = d.UpdateNodeQueues(nq.ID, map[string]interface{}{
		"akey":          nq.AgentKey,
		"nodeid":        nq.NodeId,
		"queues":        nq.Queues,
		"creation_date": nq.CreationDate,
	})

	return err
}

func (d *Database) ListNodeQueues() []dbcommon.DocItem {
	return d.ListDocs(NodeQueuesColl)
}

func (d *Database) AllNodesQueues() []queues.NodeQueues {
	return []queues.NodeQueues{}
}