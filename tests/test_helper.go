package test

import "fmt"

const (
	testUser = "deluan"
	testPassword = "wordpass"
	testClient = "test"
	testVersion = "1.0.0"
)

func AddParams(url string) string {
	return fmt.Sprintf("%s?u=%s&p=%s&c=%s&v=%s", url, testUser, testPassword, testClient, testVersion)
}


