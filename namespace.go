package settings

import "time"

type Namespace struct {
	Name    string    `json:"name,omitempty" yaml:"name,omitempty" bson:"name,omitempty"`
	Created time.Time `json:"created,omitempty" yaml:"created,omitempty" bson:"created,omitempty"`
}

func (n *Namespace) String() string {
	return n.Name
}
