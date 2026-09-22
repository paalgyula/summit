// Command dbpeek lists account names in a MongoDB database (debug helper).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	uri := flag.String("uri", "", "mongodb uri")
	db := flag.String("db", "", "database (empty = list databases)")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	c, err := mongo.Connect(ctx, options.Client().ApplyURI(*uri))
	if err != nil {
		fmt.Println("connect:", err)
		os.Exit(1)
	}
	defer func() { _ = c.Disconnect(ctx) }()

	if *db == "" {
		names, err := c.ListDatabaseNames(ctx, bson.M{})
		fmt.Println("databases:", names, "err:", err)
		for _, n := range names {
			if n == "admin" || n == "local" || n == "config" {
				continue
			}
			cols, err := c.Database(n).ListCollectionNames(ctx, bson.M{})
			fmt.Printf("  %s: %d collections %v err=%v\n", n, len(cols), cols, err)
			for _, col := range cols {
				cnt, err := c.Database(n).Collection(col).CountDocuments(ctx, bson.M{})
				fmt.Printf("    %s: %d docs err=%v\n", col, cnt, err)
			}
		}
		return
	}

	d := c.Database(*db)
	for _, col := range []string{"accounts", "characters"} {
		n, err := d.Collection(col).CountDocuments(ctx, bson.M{})
		fmt.Printf("%s.%s count=%d err=%v\n", *db, col, n, err)

		cur, err := d.Collection(col).Find(ctx, bson.M{},
			options.Find().SetProjection(bson.M{"name": 1, "email": 1}).SetLimit(20))
		if err != nil {
			fmt.Println("  find:", err)
			continue
		}
		for cur.Next(ctx) {
			var doc struct {
				Name  string `bson:"name"`
				Email string `bson:"email"`
			}
			_ = cur.Decode(&doc)
			email := "-"
			if doc.Email != "" {
				email = strings.SplitN(doc.Email, "@", 2)[0][:1] + "***"
			}
			fmt.Printf("  name=%q email=%s\n", doc.Name, email)
		}
		_ = cur.Close(ctx)
	}
}
