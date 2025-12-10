package instance


type Instance struct {
	Name       string
	Type       string
	ImageID    string
	Metadata   map[string]string
	UserData   []byte
	InstanceID string
	Status     string
}