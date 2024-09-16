package main

import (
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/function61/gokit/app/aws/lambdautils"
	"github.com/function61/gokit/app/dynversion"
	"github.com/function61/gokit/os/osutil"
	"github.com/spf13/cobra"
)

func main() {
	if lambdautils.InLambda() {
		handler, err := newServerHandler()
		osutil.ExitIfError(err)
		lambda.Start(lambdautils.NewLambdaHttpHandlerAdapter(handler))
		return
	}

	app := &cobra.Command{
		Use:     os.Args[0],
		Short:   "Dynamic DNS client & server",
		Version: dynversion.Version,
	}

	app.AddCommand(clientEntrypoint())
	app.AddCommand(serverEntrypoint())

	osutil.ExitIfError(app.Execute())
}
