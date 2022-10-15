package f860

type ErrUnauthorized struct{}

func (m *ErrUnauthorized) Error() string {
	return "Unauthorized"
}
