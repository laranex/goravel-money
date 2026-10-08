// Command setup is run by `./artisan package:install github.com/laranex/goravel-money/v4`
// (and package:uninstall). It registers the service provider in
// bootstrap/providers.go, writes config/money.go and adds MONEY_CURRENCY to
// .env.example.
package main

import (
	"os"

	"github.com/goravel/framework/packages"
	"github.com/goravel/framework/packages/modify"
	"github.com/goravel/framework/support/path"
)

func main() {
	setup := packages.Setup(os.Args)
	moduleImport := setup.Paths().Module().Import()
	provider := "&money.ServiceProvider{}"
	configPath := path.Config("money.go")
	envExample := path.Base(".env.example")

	setup.Install(
		modify.RegisterProvider(moduleImport, provider),
		modify.WhenFileNotExists(configPath,
			modify.File(configPath).Overwrite(configStub(setup.Paths().Config().Package(), setup.Paths().Facades().Import())),
		),
		modify.WhenFileExists(envExample, modify.Env(envExample, "MONEY_CURRENCY", "USD")),
	).Uninstall(
		modify.UnregisterProvider(moduleImport, provider),
		modify.WhenFileExists(configPath, modify.File(configPath).Remove()),
	).Execute()
}
