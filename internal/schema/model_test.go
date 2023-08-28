package schema

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ValidateExampleModel(t *testing.T) {
	jsonData, err := os.ReadFile("../../data/model.json")
	assert.Nil(t, err, "Error: failed reading file")

	var order Order

	decoder := json.NewDecoder(strings.NewReader(string(jsonData)))
	decoder.DisallowUnknownFields()

	err = decoder.Decode(&order)
	assert.Nil(t, err, "Error: failed unmarshalling data")

	t.Logf("%+v\n", order)
}
