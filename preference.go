package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

type Preference struct {
	Namespace string          `json:"namespace,omitempty" yaml:"namespace,omitempty" bson:"namespace,omitempty"`
	Key       string          `json:"key,omitempty" yaml:"key,omitempty" bson:"key,omitempty"`
	Value     json.RawMessage `json:"value,omitempty" yaml:"value,omitempty" bson:"value,omitempty"`
	Created   time.Time       `json:"created,omitempty" yaml:"created,omitempty" bson:"created,omitempty"`
	Updated   time.Time       `json:"updated,omitempty" yaml:"updated,omitempty" bson:"updated,omitempty"`
}

func (p *Preference) String() string {
	return fmt.Sprintf("%s.%s", p.Namespace, p.Key)
}

func (p *Preference) LoadValue(i any) error {
	decoder := json.NewDecoder(bytes.NewReader(p.Value))
	decoder.UseNumber()

	return decoder.Decode(i)
}

func (p *Preference) SetValue(i any) (err error) {
	p.Value, err = json.Marshal(i)
	return
}
