//nolint:all
// Command mongomigrate copies every collection (documents + indexes) from a
// source MongoDB database to a target one — e.g. a local dev server to a
// MongoDB Atlas cluster.
//
// Usage:
//
//	go run ./cmd/mongomigrate \
//	  -from "mongodb://admin:admin@localhost:27017" \
//	  -to   "mongodb+srv://user:pass@cluster.mongodb.net/?retryWrites=true&w=majority" \
//	  -db   summit
//
// The target collections are dropped first unless -drop=false, so the target
// ends up an exact mirror of the source. The URI is never logged.
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
	from := flag.String("from", env("SOURCE_MONGO_URI", env("MONGO_URI", "")), "source MongoDB URI")
	to := flag.String("to", env("PROD_MONGO_URI", env("ATLAS_URI", "")), "target MongoDB URI")
	fromDB := flag.String("from-db", env("MONGO_DB", "summit"), "source database")
	toDB := flag.String("to-db", env("MONGO_DB", "summit"), "target database")
	drop := flag.Bool("drop", true, "drop target collections before copying")
	batch := flag.Int("batch", 1000, "insert batch size")
	list := flag.Bool("list", false, "only list the source collections and exit")
	only := flag.String("collections", "", "comma-separated collection names (default: all)")
	timeout := flag.Duration("timeout", 15*time.Minute, "overall operation timeout")
	flag.Parse()

	if *from == "" || *to == "" {
		fatal("-from and -to (or SOURCE_MONGO_URI / PROD_MONGO_URI) are required")
	}
	if *from == *to && *fromDB == *toDB {
		fatal("source and target are identical; refusing to run")
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	src := connect(ctx, "source", *from)
	defer src.Disconnect(ctx) //nolint:errcheck
	dst := connect(ctx, "target", *to)
	defer dst.Disconnect(ctx) //nolint:errcheck

	names, err := src.Database(*fromDB).ListCollectionNames(ctx, bson.M{})
	if err != nil {
		fatal("listing source collections: %v", err)
	}
	if *only != "" {
		names = splitList(*only)
	}
	if len(names) == 0 {
		fmt.Println("source database has no collections; nothing to do")
		return
	}
	if *list {
		for _, n := range names {
			fmt.Println(n)
		}
		return
	}

	fmt.Printf("migrating %d collection(s) %q -> %q (drop=%v)\n", len(names), *fromDB, *toDB, *drop)
	for _, name := range names {
		s := src.Database(*fromDB).Collection(name)
		d := dst.Database(*toDB).Collection(name)

		if *drop {
			if err := d.Drop(ctx); err != nil {
				fatal("dropping target %s: %v", name, err)
			}
		}

		n, err := copyDocs(ctx, s, d, *batch)
		if err != nil {
			fatal("copying %s: %v", name, err)
		}
		idx, err := copyIndexes(ctx, s, d)
		if err != nil {
			fatal("copying indexes of %s: %v", name, err)
		}
		fmt.Printf("  %-28s %6d documents, %d indexes\n", name, n, idx)
	}
	fmt.Println("done")
}

func connect(ctx context.Context, label, uri string) *mongo.Client {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		fatal("connecting to %s MongoDB: %v", label, err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		fatal("pinging %s MongoDB: %v", label, err)
	}
	return client
}

// copyDocs streams every document of src into dst, preserving raw BSON.
func copyDocs(ctx context.Context, src, dst *mongo.Collection, batch int) (int, error) {
	cursor, err := src.Find(ctx, bson.M{})
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx) //nolint:errcheck

	total := 0
	buf := make([]any, 0, batch)
	flush := func() error {
		if len(buf) == 0 {
			return nil
		}
		_, err := dst.InsertMany(ctx, buf, options.InsertMany().SetOrdered(false))
		if err != nil {
			return err
		}
		total += len(buf)
		buf = buf[:0]
		return nil
	}

	for cursor.Next(ctx) {
		var doc bson.Raw
		if err := cursor.Decode(&doc); err != nil {
			return total, err
		}
		buf = append(buf, doc)
		if len(buf) >= batch {
			if err := flush(); err != nil {
				return total, err
			}
		}
	}
	if err := cursor.Err(); err != nil {
		return total, err
	}
	return total, flush()
}

// copyIndexes recreates every non-_id index of src on dst.
func copyIndexes(ctx context.Context, src, dst *mongo.Collection) (int, error) {
	cursor, err := src.Indexes().List(ctx)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx) //nolint:errcheck

	count := 0
	for cursor.Next(ctx) {
		var idx bson.M
		if err := cursor.Decode(&idx); err != nil {
			return count, err
		}
		if name, _ := idx["name"].(string); name == "_id_" {
			continue
		}
		opts := options.Index()
		if v, ok := idx["unique"].(bool); ok && v {
			opts.SetUnique(true)
		}
		if v, ok := idx["sparse"].(bool); ok && v {
			opts.SetSparse(true)
		}
		if name, ok := idx["name"].(string); ok {
			opts.SetName(name)
		}
		if _, err := dst.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: idx["key"], Options: opts}); err != nil {
			return count, err
		}
		count++
	}
	return count, cursor.Err()
}

func splitList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "mongomigrate: "+format+"\n", args...)
	os.Exit(1)
}
