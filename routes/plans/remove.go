/*

Copyright (C) 2019 Ettore Di Giacinto <mudler@gentoo.org>
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

package plans

import (
	"github.com/MottainaiCI/mottainai-server/pkg/context"
	database "github.com/MottainaiCI/mottainai-server/pkg/db"
	"github.com/MottainaiCI/mottainai-server/pkg/mottainai"
	setting "github.com/MottainaiCI/mottainai-server/pkg/settings"
	tasksapi "github.com/MottainaiCI/mottainai-server/routes/api/tasks"
)

func Delete(m *mottainai.Mottainai, ctx *context.Context, db *database.Database) {
	id := ctx.Params(":id")

	err := tasksapi.PlanDeleteById(id, db, m, ctx)
	if err != nil {
		ctx.NotFound()
		return
	}
	ctx.Invoke(func(config *setting.Config) {
		ctx.Redirect(config.GetWeb().BuildURI("/plans"))
	})
}
