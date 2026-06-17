package main

import (
	"context"
	_ "embed"

	bowtie "github.com/bowtieworks/pulumi-bowtie/provider"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/pf/tfbridge"
)

//go:embed schema.json
var schema []byte

func main() {
	meta := tfbridge.ProviderMetadata{PackageSchema: schema}
	tfbridge.Main(context.Background(), "bowtie", bowtie.Provider(), meta)
}
