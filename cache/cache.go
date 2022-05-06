package cache

type Cache struct {
	Name           string
	UrlScheme      string
	TimeToLiveDays int
}

func New(name string, urlScheme string, TimeToLiveDays int) Cache {
	return Cache{
		Name:           name,
		UrlScheme:      urlScheme,
		TimeToLiveDays: TimeToLiveDays,
	}
}
