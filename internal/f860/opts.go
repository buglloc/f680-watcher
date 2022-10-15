package f860

import "time"

type Option func(*Client)

func WithTimeout(timeout time.Duration) Option {
	return func(client *Client) {
		client.httpc.SetTimeout(timeout)
	}
}

func WithEncryptionKey(key *EncryptionKey) Option {
	return func(client *Client) {
		client.httpc.SetPreRequestHook(newRestySignHook(key))
	}
}

func WithDebug(debug bool) Option {
	return func(client *Client) {
		client.httpc.SetDebug(debug)
	}
}
