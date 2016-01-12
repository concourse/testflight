package helpers

import (
	"errors"
	"net/http"

	"github.com/concourse/fly/rc"
	"github.com/concourse/go-concourse/concourse"
)

func ConcourseClient(atcURL string) (concourse.Client, error) {
	httpClient, err := getAuthenticatedHttpClient(atcURL)
	if err != nil {
		return nil, err
	}

	conn, err := concourse.NewConnection(atcURL, httpClient)
	if err != nil {
		return nil, err
	}

	return concourse.NewClient(conn), nil
}

func getAuthenticatedHttpClient(atcURL string) (*http.Client, error) {
	dev, basicAuth, _, err := GetAuthMethods(atcURL)
	if err != nil {
		return nil, err
	}

	if dev {
		return nil, nil
	} else if basicAuth != nil {
		newUnauthedClient, err := rc.NewConnection(atcURL, false)
		if err != nil {
			return nil, err
		}

		client := &http.Client{
			Transport: basicAuthTransport{
				username: basicAuth.Username,
				password: basicAuth.Password,
				base:     newUnauthedClient.HTTPClient().Transport,
			},
		}

		return client, nil
	}

	return nil, errors.New("Unable to determine authentication")
}

type basicAuthTransport struct {
	username string
	password string

	base http.RoundTripper
}

func (t basicAuthTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.SetBasicAuth(t.username, t.password)
	return t.base.RoundTrip(r)
}
