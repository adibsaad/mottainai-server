/*

Copyright (C) 2017-2018  Ettore Di Giacinto <mudler@gentoo.org>
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

package nodesroute

import (
	"github.com/MottainaiCI/mottainai-server/pkg/context"
	"github.com/MottainaiCI/mottainai-server/pkg/db"
	"github.com/MottainaiCI/mottainai-server/routes/api/nodes"

	machinery "github.com/RichardKnop/machinery/v1"
	rabbithole "github.com/michaelklishin/rabbit-hole"
)

func Create(rmqc *rabbithole.Client, ctx *context.Context, rabbit *machinery.Server, db *database.Database) {
	_, err := nodesapi.Create(rmqc, ctx, rabbit, db)

	if err != nil {
		ctx.NotFound()
		return
	}

	ctx.Redirect("/nodes")
}
