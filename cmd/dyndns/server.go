package main

// Server - updates IP address of client in DNS if client asks (and is authorized to).

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/function61/gokit/encoding/jsonfile"
	"github.com/function61/gokit/net/http/ezhttp"
	"github.com/function61/gokit/net/http/httputils"
	"github.com/function61/gokit/os/osutil"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
)

type hostnameItem struct {
	cloudflareZoneID   string
	cloudflareRecordID string
}

// client sends this to update the DNS
type hostnameRecord struct {
	A string
	// TODO: add AAAA (= IPV6) support
}

func serverEntrypoint() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run the server",
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, args []string) {
			osutil.ExitIfError(
				server(osutil.CancelOnInterruptOrTerminate(nil)))
		},
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "update-token-validator-secret-generate",
		Short: "Generate new secret for update token validator",
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, args []string) {
			osutil.ExitIfError(func() error {
				tok := make([]byte, 256/8)
				if _, err := rand.Read(tok); err != nil {
					return err
				}

				fmt.Println(base64.RawURLEncoding.EncodeToString(tok))

				return nil
			}())
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "update-token-gen [hostname]",
		Short: "Generate update token for a hostname",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			osutil.ExitIfError(func() error {
				updateTokenValidator, err := newUpdateTokenValidator()
				if err != nil {
					return err
				}

				fmt.Println(updateTokenValidator.TokenFor(args[0]))

				return nil
			}())
		},
	})

	return cmd
}

func server(ctx context.Context) error {
	handler, err := newServerHandler()
	if err != nil {
		return err
	}

	srv := &http.Server{
		Handler: handler,
		Addr:    ":80",
	}

	return httputils.CancelableServer(ctx, srv, srv.ListenAndServe)
}

func newServerHandler() (http.Handler, error) {
	updateTokenValidator, err := newUpdateTokenValidator()
	if err != nil {
		return nil, err
	}

	cloudflareToken, err := osutil.GetenvRequired("CLOUDFLARE_API_TOKEN")
	if err != nil {
		return nil, err
	}

	routes := mux.NewRouter()
	routes.HandleFunc("/dyndns/api/hostname/{hostname}", func(w http.ResponseWriter, r *http.Request) {
		hostname := mux.Vars(r)["hostname"]

		if err := updateTokenValidator.ValidateUpdateToken(hostname, getBearerToken(r)); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		hostnameDetails, found := hostnameRegistry[hostname]
		if !found {
			http.NotFound(w, r)
			return
		}

		hr := hostnameRecord{}
		if err := jsonfile.UnmarshalDisallowUnknownFields(r.Body, &hr); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		endpoint := fmt.Sprintf(
			"https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s",
			hostnameDetails.cloudflareZoneID,
			hostnameDetails.cloudflareRecordID)

		// best to keep low in DynDNS context since this means "accepted downtime window"
		ttl := 5 * time.Minute

		if _, err := ezhttp.Put(r.Context(), endpoint, ezhttp.AuthBearer(cloudflareToken), ezhttp.SendJson(struct {
			Type    string `json:"type"`
			Name    string `json:"name"`
			Content string `json:"content"`
			TTL     int    `json:"ttl"`
			Proxied bool   `json:"proxied"`
		}{
			Type:    "A",
			Name:    hostname,
			Content: hr.A,
			TTL:     int(ttl.Seconds()),
			Proxied: false, // proxying would mean nothing else than HTTP can be behind the IP. currently our users use Wireguard etc.
		})); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}).Methods(http.MethodPut)

	return routes, nil
}
