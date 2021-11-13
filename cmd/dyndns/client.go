package main

// Client - asks server to update hostname's IP if DNS and public IP of the client changes.

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github.com/function61/gokit/net/http/ezhttp"
	"github.com/function61/gokit/os/osutil"
	"github.com/spf13/cobra"
)

const (
	serverBaseURL = "https://function61.com/dyndns"
)

func clientEntrypoint() *cobra.Command {
	return &cobra.Command{
		Use:   "client [hostname] [hostnameAuthToken]",
		Short: "Run the client",
		Args:  cobra.ExactArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			osutil.ExitIfError(
				client(osutil.CancelOnInterruptOrTerminate(nil),
					args[0],
					args[1]))
		},
	}
}

func client(ctx context.Context, hostname string, hostnameAuthToken string) error {
	myActualIP, err := resolveMyIP(ctx)
	if err != nil {
		return fmt.Errorf("resolveMyIP: %w", err)
	}

	ipsPerDNS, err := net.LookupIP(hostname)
	if err != nil {
		return err
	}

	// we could probably support >= 1 IPs but does DNS lookup guarantee order?
	// would there be any sense in dynamic DNS context anyways?
	if len(ipsPerDNS) != 1 {
		return fmt.Errorf("DNS lookup: %s: got %d IPs", hostname, len(ipsPerDNS))
	}

	ipPerDNS := ipsPerDNS[0]

	if myActualIP.Equal(ipPerDNS) { // happy path => already up-to-date
		return nil
	}

	log.Printf("actual (%s) vs DNS (%s) differ - updating DNS", myActualIP.String(), ipPerDNS.String())

	if _, err := ezhttp.Put(
		ctx,
		serverBaseURL+"/api/hostname/"+hostname,
		ezhttp.AuthBearer(hostnameAuthToken),
		ezhttp.SendJson(hostnameRecord{
			A: myActualIP.String(),
		})); err != nil {
		return fmt.Errorf("DNS update failed: %v", err)
	}

	log.Println("update succeeded")

	return nil
}

func resolveMyIP(ctx context.Context) (net.IP, error) {
	// operated by Cloudflare, so can be expected to stay around:
	//   https://major.io/2021/06/06/a-new-future-for-icanhazip/
	res, err := ezhttp.Get(ctx, "https://icanhazip.com/")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	ip := net.ParseIP(strings.TrimSuffix(string(body), "\n"))
	if ip == nil {
		return nil, fmt.Errorf("ParseIP failed: %s", string(body))
	}

	return ip, nil
}
