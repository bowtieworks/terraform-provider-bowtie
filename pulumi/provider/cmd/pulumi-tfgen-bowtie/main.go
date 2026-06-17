package main

import (
	bowtie "github.com/bowtieworks/pulumi-bowtie/provider"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/pf/tfgen"
)

func main() {
	tfgen.Main("bowtie", bowtie.Provider())
}
