package loaders

// Loader загружает документ с диска и возвращает извлечённый текст (UTF-8).
type Loader interface {
	Load(path string) (string, error)
}

// LoaderFunc реализует Loader через функцию.
type LoaderFunc func(path string) (string, error)

func (f LoaderFunc) Load(path string) (string, error) { return f(path) }
