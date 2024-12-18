package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/ebi-yade/ecresolve"
	"github.com/pkg/errors"
)

type CLI struct {
	ecresolve.Input

	Version VersionFlag `short:"v" help:"Show version and exit"`
	Format  string      `short:"f" help:"Output format [json, tag-only]" default:"json"`
}

func main() {
	if err := main_(); err != nil {
		slog.Error(fmt.Sprintf("%v", err))
		os.Exit(1)
	}
}

func main_() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	var cli CLI
	kong.Parse(&cli)

	// Clean up image revisions (remove leading : if present)
	for i, rev := range cli.Tags {
		cli.Tags[i] = strings.TrimPrefix(rev, ":")
	}

	foundImage, err := ecresolve.Resolve(ctx, cli.Input)
	if err != nil {
		return errors.Wrap(err, "error Resolve")
	}

	switch cli.Format {
	case "tag-only":
		fmt.Println(*foundImage.ImageId.ImageTag)
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(lowerCaseMarshaler{foundImage}); err != nil {
			return errors.Wrap(err, "error Encode")
		}
	default:
		return errors.Errorf("unsupported format: %s", cli.Format)
	}

	return nil
}

// ============================ Version Flag for Kong ============================

var Version = "(dev)"

type VersionFlag kong.VersionFlag

// BeforeReset writes the version variable and terminates with a 0 exit status.
func (v VersionFlag) BeforeReset(app *kong.Kong) error {
	fmt.Fprintf(app.Stdout, "%s %s\n", app.Model.Name, Version)
	app.Exit(0)
	return nil
}

// ============================ Trivial Code Below ============================

// lowerCodeMarshaler just generates AWS CLI Compatible JSON output
type lowerCaseMarshaler struct {
	Value interface{}
}

func (l lowerCaseMarshaler) MarshalJSON() ([]byte, error) {
	rawJSON, err := json.Marshal(l.Value)
	if err != nil {
		return nil, err
	}

	var intermediate interface{}
	if err := json.Unmarshal(rawJSON, &intermediate); err != nil {
		return nil, err
	}

	// キーを小文字に変換
	if m, ok := intermediate.(map[string]interface{}); ok {
		intermediate = toLowerCamelCase(m)
	}

	return json.Marshal(intermediate)
}

func toLowerCamelCase(input map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range input {
		newKey := strings.ToLower(k[:1]) + k[1:]

		switch x := v.(type) {
		case map[string]interface{}:
			result[newKey] = toLowerCamelCase(x)
		default:
			result[newKey] = v
		}
	}
	return result
}
