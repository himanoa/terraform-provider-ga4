// terraform-provider-ga4 のエントリポイント。
// Terraform から plugin として起動され、GA4 Admin API を叩くリソースを提供する。
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/himanoa/terraform-provider-ga4/internal/provider"
)

// version はリリース時に goreleaser の ldflags で上書きされる
var version = "dev"

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "デバッガを接続できるモードで起動する（Terraform からではなく手で起動するとき用）")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/himanoa/ga4",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err.Error())
	}
}
