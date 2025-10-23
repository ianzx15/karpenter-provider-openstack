package openstack

type Config struct {
    ImageID    string
    FlavorID   string
    NetworkIDs []string
    Zone       string
}
