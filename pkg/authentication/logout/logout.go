package logout

import (
	"context"
	"fmt"
	"github.com/pkg/browser"
	"github.com/run-ai/runai-cli/pkg/authentication/authentication-params"
	"github.com/run-ai/runai-cli/pkg/authentication/kubeconfig"
	"github.com/run-ai/runai-cli/pkg/authentication/pages"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"net/http"
	"net/url"
)

func Logout(user string) error {
	var err error
	if user == "" {
		err = kubeconfig.DeleteTokenToCurrentUser()
	} else {
		err = kubeconfig.DeleteTokenToUser(user)
	}
	if err != nil {
		return err
	}
	log.Debug("Tokens deleted")

	var params *authentication_params.AuthenticationParams
	if user == "" {
		params, err = kubeconfig.GetCurrentUserAuthenticationParams()
	} else {
		params, err = kubeconfig.GetUserAuthenticationParams(user)
	}
	if err != nil {
		return err
	}
	log.Debugf("Read authentication params from kubeConfig: %v", params)
	params, err = params.ValidateAndSetDefaultAuthenticationParams()
	if err != nil {
		return err
	}
	log.Debugf("Final authentication params: %v", params)

	switch params.AuthenticationFlow {
	case authentication_params.CodePkceBrowser:
		err = logoutUserSSOCookie(params)
	}
	log.Debug("Logout process done successfully")
	return err
}

func logoutUserSSOCookie(params *authentication_params.AuthenticationParams) error {
	log.Debug("Clear browser cache cookies")
	var eg errgroup.Group
	eg.Go(func() error { return serverLogoutWeb(params.ListenAddress) })
	eg.Go(func() error {
		redirectUrl := fmt.Sprintf("%vlogout", params.GetRedirectUrl())
		logoutUrl := fmt.Sprintf("%vv2/logout?returnTo=%v&client_id=%v", params.IssuerURL, url.QueryEscape(redirectUrl), params.ClientId)
		log.Debugf("Open browser url: %v", logoutUrl)
		return browser.OpenURL(logoutUrl)
	})

	return eg.Wait()
}

func serverLogoutWeb(server string) error {
	s := http.Server{Addr: server, Handler: nil}
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		logoutPage := pages.LogoutPageHtml
		fmt.Fprintf(w, logoutPage)
		go s.Shutdown(context.TODO())
	})
	log.Debug("Open server to redirect after browser logout")
	_ = s.ListenAndServe()
	return nil
}