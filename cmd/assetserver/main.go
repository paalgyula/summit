//nolint:all
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/paalgyula/summit/pkg/assetserver"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func main() {
	var listenAddr, assetDir, cacheDir, mpqDir, upstream string

	rootCmd := &cobra.Command{
		Use:   "assetserver",
		Short: "WoW Asset Server for web client with JIT WebP and GLB transcoding",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout}).With().Caller().Logger()

			if v := os.Getenv("ASSETSERVER_LISTEN"); v != "" && listenAddr == ":8080" {
				listenAddr = v
			}
			if v := os.Getenv("ASSETSERVER_ASSETS"); v != "" && assetDir == "client/assets" {
				assetDir = v
			}
			if v := os.Getenv("ASSETSERVER_CACHE"); v != "" && cacheDir == "" {
				cacheDir = v
			}
			if v := os.Getenv("ASSETSERVER_MPQ"); v != "" && mpqDir == "" {
				mpqDir = v
			}
			if v := os.Getenv("ASSETSERVER_UPSTREAM"); v != "" && upstream == "" {
				upstream = v
			}

			cfg := assetserver.Config{
				ListenAddr:  listenAddr,
				AssetDir:    assetDir,
				CacheDir:    cacheDir,
				MPQDir:      mpqDir,
				UpstreamURL: upstream,
			}

			srv, err := assetserver.NewServer(cfg)
			if err != nil {
				return fmt.Errorf("init asset server: %w", err)
			}

			go func() {
				if err := srv.Start(); err != nil {
					log.Fatal().Err(err).Msg("asset server failed")
				}
			}()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh

			log.Info().Msg("shutting down asset server")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return srv.Shutdown(ctx)
		},
	}

	rootCmd.Flags().StringVarP(&listenAddr, "listen", "l", ":8080", "HTTP listen address")
	rootCmd.Flags().StringVarP(&assetDir, "assets", "a", "client/assets", "path to assets directory")
	rootCmd.Flags().StringVarP(&cacheDir, "cache", "c", "", "path to cache directory (defaults to assets/.cache)")
	rootCmd.Flags().StringVarP(&mpqDir, "mpq", "m", "", "path to WoW Data/ directory containing MPQs for JIT extraction")
	rootCmd.Flags().StringVarP(&upstream, "upstream", "u", "", "asset server URL to fetch missing source files (ADT/M2/WMO/BLP) from, e.g. https://assets-summit.dev.pilab.hu")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
