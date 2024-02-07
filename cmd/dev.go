/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
)

// devCmd represents the dev command
var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "dev command",
	Long:  `dev command`,
	Run: func(cmd *cobra.Command, args []string) {

		// query GlobalVariable, print out
		var globalVariable GlobalVariable
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		err := globalCollection.FindOne(ctx, bson.M{"name": "GLOBAL"}).Decode(&globalVariable)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// init a reality/hysteria2 type nodes, append to globalVariable.ActiveGlobalNodes;
		// init vmesstls/vmessws type nodes, append to globalVariable.ClashLegacyNodes;

		globalVariable.ActiveGlobalNodes = append(globalVariable.ActiveGlobalNodes, Domain{
			Type:        "reality",
			Remark:      "team",
			Domain:      "www.google.com",
			IP:          "89.54.237.248",
			SNI:         "",
			UUID:        "",
			PATH:        "",
			SERVER_PORT: "7443",
			PASSWORD:    "",
			PUBLIC_KEY:  "",
			SHORT_ID:    "",
		})

		globalVariable.ClashLegacyNodes = append(globalVariable.ClashLegacyNodes, Domain{
			Type:        "vmesstls",
			Remark:      "cave",
			Domain:      "www.facebook.com",
			IP:          "211.129.65.7",
			SNI:         "",
			UUID:        "",
			PATH:        "",
			SERVER_PORT: "",
			PASSWORD:    "",
			PUBLIC_KEY:  "",
			SHORT_ID:    "",
		})

		// save globalVariable back to database
		ctx, cancel = context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		_, err = globalCollection.UpdateOne(ctx, bson.M{"name": "GLOBAL"}, bson.M{"$set": globalVariable})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

	},
}

func init() {
	rootCmd.AddCommand(devCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	devCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	devCmd.Flags().BoolP("toggle", "", false, "Help message for toggle")
}
