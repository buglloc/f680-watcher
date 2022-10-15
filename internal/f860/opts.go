package f860

type Option func(*Client)

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
