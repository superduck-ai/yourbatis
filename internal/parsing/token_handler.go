package parsing

type TokenHandler interface {
	HandleToken(content string) *string
}
