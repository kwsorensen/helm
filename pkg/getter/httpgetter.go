/*
Copyright 2016 The Kubernetes Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package getter

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"k8s.io/helm/pkg/tlsutil"
	"k8s.io/helm/pkg/urlutil"
	"k8s.io/helm/pkg/version"
)

//httpGetter is the efault HTTP(/S) backend handler
type httpGetter struct {
	client *http.Client
}

//Get performs a Get from repo.Getter and returns the body.
func (g *httpGetter) Get(href string) (*bytes.Buffer, error) {
	return g.get(href, "", "")
}

//Get performs a Get from repo.Getter using credentials and returns the body.
func (g *httpGetter) GetWithCredentials(href, username, password string) (*bytes.Buffer, error) {
	return g.get(href, username, password)
}

func (g *httpGetter) get(href, username, password string) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)

	// Set a helm specific user agent so that a repo server and metrics can
	// separate helm calls from other tools interacting with repos.
	req, err := http.NewRequest("GET", href, nil)
	if err != nil {
		return buf, err
	}
	req.Header.Set("User-Agent", "Helm/"+strings.TrimPrefix(version.GetVersion(), "v"))

	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return buf, err
	}
	if resp.StatusCode != 200 {
		return buf, fmt.Errorf("Failed to fetch %s : %s", href, resp.Status)
	}

	_, err = io.Copy(buf, resp.Body)
	resp.Body.Close()
	return buf, err
}

// newHTTPGetter constructs a valid http/https client as Getter
func newHTTPGetter(URL, CertFile, KeyFile, CAFile string) (Getter, error) {
	var client httpGetter
	if CertFile != "" && KeyFile != "" {
		tlsConf, err := tlsutil.NewClientTLS(CertFile, KeyFile, CAFile)
		if err != nil {
			return nil, fmt.Errorf("can't create TLS config for client: %s", err.Error())
		}
		tlsConf.BuildNameToCertificate()

		sni, err := urlutil.ExtractHostname(URL)
		if err != nil {
			return nil, err
		}
		tlsConf.ServerName = sni

		client.client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConf,
			},
		}
	} else {
		client.client = http.DefaultClient
	}
	return &client, nil
}
