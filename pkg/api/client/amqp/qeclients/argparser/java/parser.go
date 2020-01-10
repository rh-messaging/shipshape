package java

// Common arguments parser for cli-java

import "net/url"

func Url(u string) []string {
	parsedUrl, err := url.Parse(u)
	if err != nil {
		panic(err)
	}
	hostPort := u[0 : len(u)-len(parsedUrl.Path)]
	address := parsedUrl.Path
	return []string{"--broker", hostPort, "--address", address}
}
