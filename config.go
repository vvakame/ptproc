package ptproc

type Config struct {
	mapFile *MapFileDirective `yaml:"mapFile"`
}

type MapFileDirective struct {
	StartRegEx string `yaml:"startRegExp"`
	EndRegEx   string `yaml:"endRegEx"`
}
