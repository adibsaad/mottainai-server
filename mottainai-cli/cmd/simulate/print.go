/*

Copyright (C) 2017-2021  Ettore Di Giacinto <mudler@gentoo.org>
                         Daniele Rondina <geaaru@sabayonlinux.org>
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

package simulate

import (
	"fmt"

	setting "github.com/MottainaiCI/mottainai-server/pkg/settings"
	cobra "github.com/spf13/cobra"
)

func newSimulatePrintCommand(config *setting.Config) *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "print",
		Short: "Show configuration params",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {

			fmt.Println("===================================")
			fmt.Println("CONFIGURATION PARAMS:")
			fmt.Println("===================================")
			fmt.Println(config)
			fmt.Println("===================================")
			fmt.Println("")
			fmt.Println("===================================")
			fmt.Println("CONFIG SOURCE:")
			fmt.Println("===================================")
			fmt.Println("remote-config: ", config.Viper.Get("etcd-config"))
			if config.Viper.GetBool("etcd-config") {
				fmt.Println("Etcd Server: ", config.Viper.Get("etcd-endpoint"))
				fmt.Println("Etcd Keyring: ", config.Viper.Get("etcd-keyring"))
				fmt.Println("Etcd Path: ", config.Viper.Get("config"))
			} else {
				fmt.Println("Config File: ", config.Viper.Get("config"))
			}
			fmt.Println("===================================")
		},
	}

	return cmd
}
