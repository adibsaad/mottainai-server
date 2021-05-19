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

package scheduler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	setting "github.com/MottainaiCI/mottainai-server/mottainai-scheduler/pkg/config"
	specs "github.com/MottainaiCI/mottainai-server/mottainai-scheduler/pkg/specs"
	"github.com/MottainaiCI/mottainai-server/pkg/client"
	"github.com/MottainaiCI/mottainai-server/pkg/nodes"
	"github.com/MottainaiCI/mottainai-server/pkg/queues"
	msetting "github.com/MottainaiCI/mottainai-server/pkg/settings"
	schema "github.com/MottainaiCI/mottainai-server/routes/schema"
	v1 "github.com/MottainaiCI/mottainai-server/routes/schema/v1"

	"github.com/mudler/anagent"
)

type DefaultTaskScheduler struct {
	Config    *setting.Config
	Scheduler *anagent.Anagent
	Fetcher   client.HttpClient

	Mutex sync.Mutex

	DefaultQueue string
	Agents       map[string]nodes.Node
}

func NewDefaultTaskScheduler(config *setting.Config, agent *anagent.Anagent) *DefaultTaskScheduler {
	ans := &DefaultTaskScheduler{
		Config:       config,
		Scheduler:    agent,
		DefaultQueue: "general",
		Agents:       []nodes.Node{},
		Mutex:        sync.Mutex{},
	}

	// Initialize fetcher
	fetcher := client.NewTokenClient(
		config.GetWeb().AppURL,
		config.GetScheduler().ApiKey,
		config.ToMottainaiConfig(),
	)

	ans.Fetcher = fetcher

	return ans
}

func (s *DefaultTaskScheduler) RetrieveDefaultQueue() error {
	var tlist []msetting.Setting

	req := &schema.Request{
		Route:  v1.Schema.GetSettingRoute("show_all"),
		Target: &tlist,
	}

	err := s.Fetcher.Handle(req)
	if err != nil {
		return err
	}

	// Retrieve general queue config
	for _, i := range tlist {
		if i.Key == msetting.SYSTEM_TASKS_DEFAULT_QUEUE {
			s.DefaultQueue = i.Value
			break
		}
	}

	return nil
}

func (s *DefaultTaskScheduler) RetrieveNodes() error {
	var (
		n        []nodes.Node
		filtered []nodes.Node
	)

	req := &schema.Request{
		Route:  v1.Schema.GetNodeRoute("show_all"),
		Target: &n,
	}

	err := s.Fetcher.Handle(req)
	if err != nil {
		return err
	}

	if len(s.Config.GetScheduler().Queues) > 0 {
		for _, node := range n {
			valid := false
			for _, q := range s.Config.GetScheduler().Queues {
				if node.HasQueue(q) {
					valid = true
					break
				}
			}
			if valid {
				filtered = append(filtered, node)
			}
		}
		n = filtered
	}

	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	m := make(map[string]nodes.Node, 0)

	for _, node := range n {
		m[fmt.Sprintf("%s-%s", node.NodeID, node.Key)] = node
	}
	// Store agent data
	s.Agents = m

	return nil
}

func (s *DefaultTaskScheduler) GetQueues() ([]queues.Queue, error) {
	var n []queues.Queue

	req := &schema.Request{
		Route:  v1.Schema.GetQueueRoute("show_all"),
		Target: &n,
	}

	if len(s.Config.GetScheduler().Queues) > 0 {
		// POST: Get only defined queues
		body := map[string]interface{}{
			"queues": s.Config.GetScheduler().Queues,
		}

		b, err := json.Marshal(body)
		if err != nil {
			return n, err
		}

		req.Body = bytes.NewBuffer(b)
	}

	err := s.Fetcher.Handle(req)
	if err != nil {
		return n, err
	}

	return n, nil
}

func (s *DefaultTaskScheduler) GetNodeQueues() ([]queues.NodeQueues, error) {
	var (
		n        []queues.NodeQueues
		filtered []queues.NodeQueues
	)

	// TODO: Add filter options
	req := &schema.Request{
		Route:  v1.Schema.GetNodeQueueRoute("show_all"),
		Target: &n,
	}

	err := s.Fetcher.Handle(req)
	if err != nil {
		return n, err
	}

	// Get The queues of the nodes available
	for _, q := range n {
		if _, ok := s.Agents[fmt.Sprintf("%s-%s", q.NodeId, q.AgentKey)]; ok {
			filtered = append(filtered, q)
		}
	}

	return filtered, nil
}

func (s *DefaultTaskScheduler) Schedule() error {
	tasksmap, err := s.GetTasks2Inject()
	if err != nil {
		return err
	}

	if len(taskmap) > 0 {
		for nodeid, m := range tasksmap {

			akey := s.Agents[nodeid].Key
			nid := s.Agents[nodeid].NodeID

			for q, tasks := range m {
				fields := strings.Split(q, "|")
				// fields[0] contains queue name
				// fields[1] contains queue id
				for _, tid := range tasks {
					_, err := s.Fetcher.NodeQueueAddTask(akey, nid, fields[0], tid)
					if err != nil {
						return err
					}

					// Remote task from queue to avoid reinjection
					req := &schema.Request{
						Route: v1.Schema.GetQueueRoute("del_task"),
						Options: map[string]interface{}{
							":qid": fields[1],
							":tid": tid,
						},
					}
					resp, err := s.Fetcher.HandleAPIResponse(req)
					if err != nil {
						return err
					}
					if resp.Status == "ko" {
						return errors.New("Error on delete task " + tid + " from queue " + fields[0])
					}
				}
			}
		}
	}

	return nil
}

func (s *DefaultTaskScheduler) GetTasks2Inject() (map[string]map[string][]string, error) {
	ans := make(map[string]map[string][]string, 0)

	queues, err := s.GetQueues()
	if err != nil {
		return ans, err
	}

	queuesWithTasks := []queues.Queue{}
	// Identify queues with tasks in waiting
	for _, q := range queues {
		if len(q.Waiting) > 0 {
			queuesWithTasks = append(queuesWithTasks, q)
		}
	}

	if len(queuesWithTasks) == 0 {
		// Nothing to do
		return ans, nil
	}

	// Retrieve node quests
	nodeQueues, err := s.GetNodeQueues()
	if err != nil {
		return ans, err
	}

	for _, q := range queuesWithTasks {
		m, err := s.elaborateQueue(q, nodeQueues)
		if err != nil {
			fmt.Println(fmt.Sprintf("Error on elaborate queue %s: %s",
				q, err.Error()))
			continue
		}

		if len(m) > 0 {
			for idnode, tasks := range m {
				ans[idnode][fmt.Sprintf("%s|%s", q.Name, q.Qid)] = tasks
			}
		}
	}

	return ans, nil
}

func (s *DefaultTaskScheduler) elaborateQueue(queue queues.Queue, nodeQueues []queues.NodeQueues) (map[string][]string, error) {
	ans := make(map[string][]string, 0)

	// Retrieve the list of agents with the specified queues
	validAgents := []specs.NodeSlots{}
	for _, node := range nodeQueues {
		nodeKey := fmt.Sprintf("%s-%s", node.NodeId, node.AgentKey)
		if maxTasks, ok := s.Agents[nodeKey].Queues[queue.Name]; ok {
			tt, ok := node.Queues[queue.Name]
			slot := specs.NodeSlots{
				Key: nodeKey,
			}

			if !ok {
				slot.AvailableSlot = maxTasks
				validAgents = append(validAgents, slot)
			} else if len(tt) < maxTasks {
				slot.AvailableSlot = maxTasks - len(tt)
				validAgents = append(validAgents, slot)
			}
		}
	}

	if len(validAgents) > 0 {
		sort.Sort(specs.NodeSlotsList(validAgents))

		for _, node := range validAgents {
			if len(queue.Waiting) == 0 {
				break
			}

			for ; node.AvailableSlot > 0; node.AvailableSlot-- {
				if len(queue.Waiting) == 0 {
					break
				}
				tid := queue.Waiting[0]
				queue.Waiting = queue.Waiting[1:]
				if _, ok := ans[node.Key]; ok {
					ans[node.Key] = append(ans[node.Key], tid)
				} else {
					ans[node.Key] = []string{tid}
				}
			}
		}
	}

	fmt.Println("FOR QUEUE ", queue, " : ", ans)

	return ans, nil
}