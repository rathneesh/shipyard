package dockerhub

type Tag struct {
	Layer string `json:"layer"`
	Name  string `json:"name"`
}
type TagV2 struct {
	Name string `json:"name"`
	Tags []string `json:"tags"`
}
