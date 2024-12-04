package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xianml/s0cmd/internal/download"
)

var (
	// Version information
	Version   = "v0.1.0"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "s0cmd",
	Short: "s0cmd is a tool for downloading objects from S3-compatible APIs",
	Long:  `s0cmd allows you to download ML models and other objects from S3-compatible storage with high speed.`,
}

var getCmd = &cobra.Command{
	Use:   "get [s3 presigned URL]",
	Short: "Download an object from S3",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		s3URL := args[0]
		outputFile, _ := cmd.Flags().GetString("output")
		parallelism, _ := cmd.Flags().GetInt("parallelism")

		fmt.Printf("Downloading %s to %s with %d parallelism...\n", s3URL, outputFile, parallelism)
		d := download.Downloader{
			Parallelism: parallelism,
			Output:      outputFile,
		}
		if err := d.Download(cmd.Context(), s3URL); err!=nil{
			fmt.Println("Failed with", err.Error())
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("s0cmd %s\n", Version)
		fmt.Printf("Git commit: %s\n", GitCommit)
		fmt.Printf("Built: %s\n", BuildTime)
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(versionCmd)
	getCmd.Flags().StringP("output", "o", "output.file", "Output file name")
	getCmd.Flags().IntP("parallelism", "p", 4, "Number of parallel downloads")
}

func Execute(ctx context.Context) error {
	_, err := rootCmd.ExecuteContextC(ctx)
	return err
}
